package uback

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/sirupsen/logrus"
)

var (
	backupFilenameRe   = regexp.MustCompile(fmt.Sprintf(`^(%s)-(full|from-(%s))(\.ubkp)?$`, SnapshotRe, SnapshotRe))
	SnapshotRe         = `\d{8}T\d{6}\.\d{3}`  // Regexp matching a snapshot name
	SnapshotTimeFormat = "20060102T150405.000" // Time format of a snapshot, for time.Parse / time.Format
)

// Part of RetentionPolicySubject interface
func (s Snapshot) Time() (time.Time, error) {
	return time.ParseInLocation(SnapshotTimeFormat, string(s), time.UTC)
}

// Part of RetentionPolicySubject interface
func (s Snapshot) IsFull() bool {
	return true
}

// Part of RetentionPolicySubject interface
func (s Snapshot) Name() string {
	return string(s)
}

// Part of RetentionPolicySubject interface
func (b Backup) Time() (time.Time, error) {
	return b.Snapshot.Time()
}

// Part of RetentionPolicySubject interface
func (b Backup) IsFull() bool {
	return b.BaseSnapshot == nil
}

// Part of RetentionPolicySubject interface
func (b Backup) Name() string {
	return b.Snapshot.Name()
}

// Name of a backup: (Snapshot.Name() + "-full") for a full backup, (Snapshot.Name() + "-from-" + BaseSnapshot.Name()) for an incremental backup
func (b Backup) FullName() string {
	if b.BaseSnapshot == nil {
		return fmt.Sprintf("%s-full", b.Snapshot.Name())
	}
	return fmt.Sprintf("%s-from-%s", b.Snapshot.Name(), b.BaseSnapshot.Name())
}

// Shorthand for Fullname() + ".ubkp"
func (b Backup) Filename() string {
	return b.FullName() + ".ubkp"
}

// Compare backups by the date of their snapshot
func CompareBackups(a, b Backup) int {
	return CompareSnapshots(a.Snapshot, b.Snapshot)
}

// Compare snapshots by date
func CompareSnapshots(a, b Snapshot) int {
	return strings.Compare(string(a), string(b))
}

// Sorted from most recent to least recent
func SortedListBackups(dst Destination) ([]Backup, error) {
	backups, err := dst.ListBackups()
	if err != nil {
		return nil, err
	}

	sort.Slice(backups, func(a, b int) bool {
		return CompareBackups(backups[a], backups[b]) >= 0
	})

	return backups, nil
}

// Sorted from most recent to least recent
func SortedListArchives(src Source) ([]Snapshot, error) {
	archives, err := src.ListArchives()
	if err != nil {
		return nil, err
	}

	sort.Slice(archives, func(a, b int) bool {
		return CompareSnapshots(archives[a], archives[b]) >= 0
	})

	return archives, nil
}

func SortedListBookmarks(src Source) ([]Snapshot, error) {
	bookmarks, err := src.ListBookmarks()
	if err != nil {
		return nil, err
	}

	sort.Slice(bookmarks, func(a, b int) bool {
		return CompareSnapshots(bookmarks[a], bookmarks[b]) >= 0
	})

	return bookmarks, nil
}

// Reverse of Backup.Filename()
func ParseBackupFilename(f string, requireExt bool) (Backup, error) {
	f = path.Base(f)
	m := backupFilenameRe.FindStringSubmatch(f)
	if m == nil {
		return Backup{}, fmt.Errorf("cannot parse backup filename: %s", f)
	}

	if requireExt && m[4] != ".ubkp" {
		return Backup{}, fmt.Errorf("cannot parse backup filename: %s: missing or invalid extension '%s'", f, m[4])
	}

	if m[2] == "full" {
		return Backup{Snapshot: Snapshot(m[1])}, nil
	}

	baseSnapshot := Snapshot(m[3])
	return Backup{Snapshot: Snapshot(m[1]), BaseSnapshot: &baseSnapshot}, nil
}

// This should really be in the standard library...
func CopyFile(dst, src string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()

	st, err := srcF.Stat()
	if err != nil {
		return err
	}

	dstF, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, st.Mode())
	if err != nil {
		return err
	}
	defer dstF.Close()

	_, err = io.Copy(dstF, srcF)
	return err
}

// Intended to be used in a source CreateBackup(). If the created backup data is simply given by a command
// stdout, make a ReadCloser from an exec.Command stdout. When the subprocess is done, call finalize with
// the result of cmd.Wait() as an argument. The result of finalize will be the error returned by the reader
// on the next read/close, until the error is nil (then EOF will be returned on next read)
func WrapSourceCommand(backup Backup, cmd *exec.Cmd, finalize func(err error) error) (Backup, io.ReadCloser, error) {
	logrus.Printf("running: %v", cmd.String())

	pr, pw := io.Pipe()

	cmd.Stdin = nil
	cmd.Stdout = pw
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return Backup{}, nil, err
	}

	go func() {
		if finalize == nil {
			pw.CloseWithError(cmd.Wait())
		} else {
			pw.CloseWithError(finalize(cmd.Wait()))
		}
	}()

	return backup, pr, nil
}

// Index a list of backups by the name of the backup snapshot
func MakeIndex(backups []Backup) map[string]Backup {
	idx := make(map[string]Backup)
	for _, b := range backups {
		idx[b.Name()] = b
	}
	return idx
}

// Create the dependencies chain from a backup:
//   - for a full backup, the chain consists only of the backup itself
//   - for an incremental backup, the chain consists of the backup itself and its base's chain
func GetFullChain(backup Backup, index map[string]Backup) ([]Backup, bool) {
	ok := true
	chain := []Backup{backup}
	for !backup.IsFull() {
		backup, ok = index[backup.BaseSnapshot.Name()]
		if !ok {
			break
		}
		chain = append(chain, backup)
	}
	return chain, ok
}

// Load a private key either from a file (if keyFile argument is provided), or from its content (key argument)
func LoadIdentities(keyFile, key string) ([]age.Identity, error) {
	if keyFile != "" && key != "" {
		return nil, fmt.Errorf("must provide one of key file or key, not both")
	}

	if keyFile != "" {
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}

		key = string(keyData)
	}

	return age.ParseIdentities(bytes.NewBufferString(key))
}

// Load a public key either from a file (if keyFile argument is provided), or from its content (key argument)
// If the file or the content represents a private key, derive the public key from it
func LoadRecipients(keyFile, key string) ([]age.Recipient, error) {
	if keyFile != "" && key != "" {
		return nil, fmt.Errorf("must provide one of key file or key, not both")
	}

	if keyFile != "" {
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}

		key = string(keyData)
	}

	return age.ParseRecipients(bytes.NewBufferString(key))
}

func BuildCommand(command []string, additionalArgs ...string) *exec.Cmd {
	fullArgs := append(append([]string{}, command...), additionalArgs...)
	cmd := exec.Command(fullArgs[0], fullArgs[1:]...)
	cmd.Stdout = os.Stderr // default stdout to stderr because we don't want other processes to output stuff on our output
	cmd.Stderr = os.Stderr
	return cmd
}

func StartCommand(log *logrus.Entry, cmd *exec.Cmd) error {
	log.Printf("starting: %s", cmd.String())
	return cmd.Start()
}

func RunCommand(log *logrus.Entry, cmd *exec.Cmd) error {
	log.Printf("starting: %s", cmd.String())
	return cmd.Run()
}
