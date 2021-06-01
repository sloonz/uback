package uback

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

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
func SortedListSnapshots(src Source) ([]Snapshot, error) {
	snapshots, err := src.ListSnapshots()
	if err != nil {
		return nil, err
	}

	sort.Slice(snapshots, func(a, b int) bool {
		return CompareSnapshots(snapshots[a], snapshots[b]) >= 0
	})

	return snapshots, nil
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
//  * for a full backup, the chain consists only of the backup itself
//  * for an incremental backup, the chain consists of the backup itself and its base's chain
func GetFullChain(backups []Backup, backup Backup, index map[string]Backup) ([]Backup, bool) {
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

func BuildCommand(command []string, additionalArgs ...string) *exec.Cmd {
	fullArgs := make([]string, 0, len(command)+len(additionalArgs)-1)
	fullArgs = append(fullArgs, command[1:]...)
	fullArgs = append(fullArgs, additionalArgs...)
	return exec.Command(command[0], fullArgs...)
}
