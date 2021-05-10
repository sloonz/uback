package sources

import (
	"github.com/sloonz/uback/lib"

	_ "embed" // required for go:embed
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrMariaBackupCommand = errors.New("mariabackup source: missing or invalid command")
	mariaBackupLog        = logrus.WithFields(logrus.Fields{
		"source": "mariabackup",
	})

	//go:embed scripts/sqldump-docker.sh
	mariaBackupDockerScript []byte

	//go:embed scripts/sqldump-local.sh
	mariaBackupLocalScript []byte
)

type mariaBackupSource struct {
	options       *uback.Options
	snapshotsPath string
	command       []string
	authFileData  string
}

func newMariaBackupSource(options *uback.Options) (uback.Source, error) {
	snapshotsPath := options.String["SnapshotsPath"]
	if snapshotsPath == "" {
		mariaBackupLog.Warnf("SnapshotsPath option missing, incremental backups will not impossible")
	} else {
		err := os.MkdirAll(snapshotsPath, 0777)
		if err != nil {
			return nil, err
		}
	}

	command := options.GetCommand("Command", []string{"mariabackup"})
	if len(command) == 0 {
		return nil, ErrMariaBackupCommand
	}

	authFileData := ""
	user, hasUser := options.String["User"]
	pwd, hasPassword := options.String["Password"]
	if hasUser || hasPassword {
		authFileData = "[mariabackup]\n"
		if hasUser {
			authFileData += fmt.Sprintf("user=%s\n", user)
		}
		if hasPassword {
			authFileData += fmt.Sprintf("password=%s\n", pwd)
		}
	}

	return &mariaBackupSource{options: options, snapshotsPath: snapshotsPath, command: command, authFileData: authFileData}, nil
}

func newMariaBackupSourceForRestoration() (uback.Source, error) {
	return &mariaBackupSource{}, nil
}

// Part of uback.Source interface
func (s *mariaBackupSource) ListSnapshots() ([]uback.Snapshot, error) {
	if s.snapshotsPath == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(s.snapshotsPath)
	if err != nil {
		return nil, err
	}

	var snapshots []uback.Snapshot

	re := regexp.MustCompile(fmt.Sprintf("^%s$", uback.SnapshotRe))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") || !entry.IsDir() {
			continue
		}

		if !re.MatchString(entry.Name()) {
			mariaBackupLog.WithFields(logrus.Fields{"file": entry.Name()}).Warnf("invalid snapshot name")
			continue
		}

		snapshots = append(snapshots, uback.Snapshot(entry.Name()))
	}

	return snapshots, nil
}

// Part of uback.Source interface
func (s *mariaBackupSource) RemoveSnapshot(snapshot uback.Snapshot) error {
	if s.snapshotsPath == "" {
		return nil
	}
	return os.RemoveAll(path.Join(s.snapshotsPath, string(snapshot)))
}

// Part of uback.Source interface
func (s *mariaBackupSource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	snapshot := time.Now().UTC().Format(uback.SnapshotTimeFormat)
	tmpSnapshotPath := path.Join(s.snapshotsPath, fmt.Sprintf("_tmp-%s", snapshot))
	finalSnapshotPath := path.Join(s.snapshotsPath, snapshot)

	command := make([]string, 0, len(s.command)+5)
	command = append(command, s.command...)

	var authFile *os.File
	var err error
	if s.authFileData != "" {
		authFile, err = os.CreateTemp(s.snapshotsPath, "")
		if err != nil {
			return uback.Backup{}, nil, err
		}
		_, err = authFile.Write([]byte(s.authFileData))
		if err != nil {
			return uback.Backup{}, nil, err
		}
		err = authFile.Sync()
		if err != nil {
			return uback.Backup{}, nil, err
		}
		command = append(command, fmt.Sprintf("--defaults-file=%s", authFile.Name()))
	}

	command = append(command, "--backup")
	command = append(command, "--stream=mbstream")
	if s.snapshotsPath != "" {
		command = append(command, fmt.Sprintf("--extra-lsndir=%s", tmpSnapshotPath))
		if baseSnapshot != nil {
			command = append(command, fmt.Sprintf("--incremental-basedir=%s", path.Join(s.snapshotsPath, baseSnapshot.Name())))
		}
	} else {
		baseSnapshot = nil
	}

	backup := uback.Backup{Snapshot: uback.Snapshot(snapshot), BaseSnapshot: baseSnapshot}
	mariaBackupLog.Printf("creating backup: %s", backup.Filename())

	return uback.WrapSourceCommand(backup, exec.Command(command[0], command[1:]...), func(err error) error {
		if authFile != nil {
			n := authFile.Name()
			authFile.Close()
			os.Remove(n)
		}
		if err != nil {
			if s.snapshotsPath != "" {
				os.RemoveAll(tmpSnapshotPath)
			}
			return err
		}
		if s.snapshotsPath != "" {
			return os.Rename(tmpSnapshotPath, finalSnapshotPath)
		}
		return nil

	})
}

// Part of uback.Source interface
func (s *mariaBackupSource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	err := os.RemoveAll(path.Join(targetDir, backup.Snapshot.Name()))
	if err != nil {
		return err
	}

	restoreDir := path.Join(targetDir, backup.Snapshot.Name())
	err = os.MkdirAll(restoreDir, 0777)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(restoreDir, "sqldump-docker.sh"), mariaBackupDockerScript, 0777)
	if err != nil {
		mariaBackupLog.Warnf("cannot write sqldump-docker.sh script: %v", err)
	}

	err = os.WriteFile(path.Join(restoreDir, "sqldump-local.sh"), mariaBackupLocalScript, 0777)
	if err != nil {
		mariaBackupLog.Warnf("cannot write sqldump-local.sh script: %v", err)
	}

	cmd := exec.Command("mbstream", "-x", "-C", restoreDir)
	cmd.Stdin = data
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	mariaBackupLog.Printf("running %v", cmd.String())
	err = cmd.Run()
	if err != nil {
		return err
	}

	if backup.BaseSnapshot != nil {
		baseDir := path.Join(targetDir, backup.BaseSnapshot.Name())
		cmd = exec.Command("mariabackup", "--prepare", fmt.Sprintf("--target-dir=%s", baseDir), fmt.Sprintf("--incremental-dir=%s", restoreDir))
		cmd.Stdin = nil
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		mariaBackupLog.Printf("running %v", cmd.String())
		err = cmd.Run()
		if err != nil {
			return err
		}

		err = os.RemoveAll(restoreDir)
		if err != nil {
			return err
		}

		return os.Rename(baseDir, restoreDir)
	}

	cmd = exec.Command("mariabackup", "--prepare", fmt.Sprintf("--target-dir=%s", restoreDir))
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	mariaBackupLog.Printf("running %v", cmd.String())
	return cmd.Run()
}
