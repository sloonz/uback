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
	ErrBtrfsPath = errors.New("btrfs source: invalid path")
	btrfsLog     = logrus.WithFields(logrus.Fields{
		"source": "btrfs",
	})
)

type btrfsSource struct {
	options         *uback.Options
	snapshotsPath   string
	basePath        string
	snapshotCommand []string
	sendCommand     []string
	receiveCommand  []string
	deleteCommand   []string
}

func newBtrfsSource(options *uback.Options) (uback.Source, error) {
	snapshotsPath := options.String["SnapshotsPath"]
	if snapshotsPath == "" {
		btrfsLog.Warnf("SnapshotsPath option missing, incremental backups will not impossible")
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

	return &btrfsSource{
		options:         options,
		snapshotsPath:   snapshotsPath,
		basePath:        basePath,
		snapshotCommand: options.GetCommand("SnapshotCommand", []string{"btrfs", "subvolume", "snapshot"}),
		sendCommand:     options.GetCommand("SendCommand", []string{"btrfs", "send"}),
		deleteCommand:   options.GetCommand("DeleteCommand", []string{"btrfs", "subvolume", "delete"}),
	}, nil
}

func newBtrfsSourceForRestoration(options *uback.Options) (uback.Source, error) {
	return &btrfsSource{receiveCommand: options.GetCommand("ReceiveCommand", []string{"btrfs", "receive"})}, nil
}

// Part of uback.Source interface
func (s *btrfsSource) ListSnapshots() ([]uback.Snapshot, error) {
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
			btrfsLog.WithFields(logrus.Fields{"file": entry.Name()}).Warnf("invalid snapshot name")
			continue
		}

		snapshots = append(snapshots, uback.Snapshot(entry.Name()))
	}

	return snapshots, nil
}

// Part of uback.Source interface
func (s *btrfsSource) RemoveSnapshot(snapshot uback.Snapshot) error {
	if s.snapshotsPath == "" {
		return nil
	}
	args := []string{}
	args = append(args, s.deleteCommand...)
	args = append(args, path.Join(s.snapshotsPath, string(snapshot)))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	btrfsLog.Printf("running %s", cmd.String())
	return cmd.Run()
}

// Part of uback.Source interface
func (s *btrfsSource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	snapshot := time.Now().UTC().Format(uback.SnapshotTimeFormat)
	tmpSnapshotPath := path.Join(s.snapshotsPath, fmt.Sprintf("_tmp-%s", snapshot))
	finalSnapshotPath := path.Join(s.snapshotsPath, snapshot)

	if s.snapshotsPath == "" {
		baseSnapshot = nil
	}

	args := []string{}
	args = append(args, s.snapshotCommand...)
	args = append(args, "-r", s.basePath, tmpSnapshotPath)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	btrfsLog.Printf("running %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		return uback.Backup{}, nil, err
	}

	backup := uback.Backup{Snapshot: uback.Snapshot(snapshot), BaseSnapshot: baseSnapshot}
	btrfsLog.Printf("creating backup: %s", backup.Filename())

	args = nil
	args = append(args, s.sendCommand...)
	if baseSnapshot != nil {
		args = append(args, "-p", path.Join(s.snapshotsPath, baseSnapshot.Name()))
	}
	args = append(args, tmpSnapshotPath)
	return uback.WrapSourceCommand(backup, exec.Command(args[0], args[1:]...), func(err error) error {
		if err != nil || s.snapshotsPath == "" {
			args = nil
			args = append(args, s.deleteCommand...)
			args = append(args, tmpSnapshotPath)
			_ = exec.Command(args[0], args[1:]...).Run()
			return err
		}
		if s.snapshotsPath != "" {
			return os.Rename(tmpSnapshotPath, finalSnapshotPath)
		}
		return nil
	})
}

// Part of uback.Source interface
func (s *btrfsSource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	args := []string{}
	args = append(args, s.receiveCommand...)
	args = append(args, targetDir)
	cmd := exec.Command(args[0], args[1:]...)
	btrfsLog.Printf("running %s", cmd.String())
	cmd.Stdin = data
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return nil
	}

	return os.Rename(path.Join(targetDir, "_tmp-"+backup.Snapshot.Name()), path.Join(targetDir, backup.Snapshot.Name()))
}
