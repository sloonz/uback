package sources

import (
	"github.com/sloonz/uback/lib"

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
	ErrTarPath = errors.New("tar source: invalid path")
	tarLog     = logrus.WithFields(logrus.Fields{
		"source": "tar",
	})
)

type tarSource struct {
	options       *uback.Options
	snapshotsPath string
	basePath      string
	command       []string
}

func newTarSource(options *uback.Options) (uback.Source, error) {
	snapshotsPath := options.String["SnapshotsPath"]
	if snapshotsPath == "" {
		tarLog.Warnf("SnapshotsPath option missing, incremental backups will not impossible")
	} else {
		err := os.MkdirAll(snapshotsPath, 0777)
		if err != nil {
			return nil, err
		}
	}

	basePath := options.String["Path"]
	if basePath == "" {
		return nil, ErrTarPath
	}

	command := options.GetCommand("Command", []string{"tar"})

	return &tarSource{options: options, snapshotsPath: snapshotsPath, basePath: basePath, command: command}, nil
}

func newTarSourceForRestoration() (uback.Source, error) {
	return &tarSource{}, nil
}

// Part of uback.Source interface
func (s *tarSource) ListSnapshots() ([]uback.Snapshot, error) {
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
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") || entry.IsDir() {
			continue
		}

		if !re.MatchString(entry.Name()) {
			tarLog.WithFields(logrus.Fields{"file": entry.Name()}).Warnf("invalid snapshot name")
			continue
		}

		snapshots = append(snapshots, uback.Snapshot(entry.Name()))
	}

	return snapshots, nil
}

// Part of uback.Source interface
func (s *tarSource) RemoveSnapshot(snapshot uback.Snapshot) error {
	if s.snapshotsPath == "" {
		return nil
	}
	return os.Remove(path.Join(s.snapshotsPath, string(snapshot)))
}

// Part of uback.Source interface
func (s *tarSource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	snapshot := time.Now().UTC().Format(uback.SnapshotTimeFormat)
	tmpSnapshotPath := path.Join(s.snapshotsPath, fmt.Sprintf("_tmp-%s", snapshot))
	finalSnapshotPath := path.Join(s.snapshotsPath, snapshot)

	if s.snapshotsPath == "" {
		baseSnapshot = nil
	}

	if baseSnapshot != nil {
		err := uback.CopyFile(tmpSnapshotPath, path.Join(s.snapshotsPath, baseSnapshot.Name()))
		if err != nil {
			tarLog.WithFields(logrus.Fields{"snapshot": baseSnapshot.Name()}).Warnf("failed to copy base snapshot (%v), forcing full backup", err)
			os.Remove(tmpSnapshotPath)
			baseSnapshot = nil
		}
	}

	backup := uback.Backup{Snapshot: uback.Snapshot(snapshot), BaseSnapshot: baseSnapshot}
	tarLog.Printf("creating backup: %s", backup.Filename())

	args := []string{"--create", "-C", s.basePath}
	if s.snapshotsPath != "" {
		args = append(args, fmt.Sprintf("--listed-incremental=%s", tmpSnapshotPath))
	}
	args = append(args, ".")
	return uback.WrapSourceCommand(backup, uback.BuildCommand(s.command, args...), func(err error) error {
		// For tar, exit code 1 is a warning, don't treat it as an error
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ProcessState.ExitCode() != 1 {
				if s.snapshotsPath != "" {
					os.Remove(tmpSnapshotPath)
				}
				return err
			}
		}
		if s.snapshotsPath != "" {
			return os.Rename(tmpSnapshotPath, finalSnapshotPath)
		}
		return nil
	})
}

// Part of uback.Source interface
func (s *tarSource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	err := os.RemoveAll(path.Join(targetDir, backup.Snapshot.Name()))
	if err != nil {
		return err
	}

	if backup.BaseSnapshot == nil {
		err = os.MkdirAll(path.Join(targetDir, backup.Snapshot.Name()), 0777)
	} else {
		err = os.Rename(path.Join(targetDir, backup.BaseSnapshot.Name()), path.Join(targetDir, backup.Snapshot.Name()))
	}
	if err != nil {
		return err
	}

	cmd := exec.Command("tar", "-x", "-C", path.Join(targetDir, backup.Snapshot.Name()))
	cmd.Stdin = data
	return uback.RunCommand(tarLog, cmd)
}
