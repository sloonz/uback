package sources

import (
	"github.com/sloonz/uback/lib"

	"errors"
	"fmt"
	"io"
	"os"
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
	reuseSnapshots  int
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

	var reuseSnapshots int
	if options.String["ReuseSnapshots"] != "" {
		var err error
		reuseSnapshots, err = uback.ParseInterval(options.String["ReuseSnapshots"])
		if err != nil {
			return nil, err
		}
	}

	return &btrfsSource{
		options:         options,
		snapshotsPath:   snapshotsPath,
		basePath:        basePath,
		snapshotCommand: options.GetCommand("SnapshotCommand", []string{"btrfs", "subvolume", "snapshot"}),
		sendCommand:     options.GetCommand("SendCommand", []string{"btrfs", "send"}),
		deleteCommand:   options.GetCommand("DeleteCommand", []string{"btrfs", "subvolume", "delete"}),
		reuseSnapshots:  reuseSnapshots,
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
	cmd := uback.BuildCommand(s.deleteCommand, path.Join(s.snapshotsPath, string(snapshot)))
	return uback.RunCommand(btrfsLog, cmd)
}

// Part of uback.Source interface
func (s *btrfsSource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	snapshot := time.Now().UTC().Format(uback.SnapshotTimeFormat)
	finalSnapshotPath := path.Join(s.snapshotsPath, snapshot)
	tmpSnapshotPath := path.Join(s.snapshotsPath, fmt.Sprintf("_tmp-%s", snapshot))

	if s.reuseSnapshots != 0 {
		snapshots, err := uback.SortedListSnapshots(s)
		if err != nil {
			return uback.Backup{}, nil, err
		}

		if len(snapshots) > 0 {
			t, err := snapshots[0].Time()
			if err != nil {
				return uback.Backup{}, nil, err
			}

			if time.Now().UTC().Sub(t).Seconds() <= float64(s.reuseSnapshots) {
				snapshot = string(snapshots[0])
				finalSnapshotPath = path.Join(s.snapshotsPath, snapshot)
				tmpSnapshotPath = finalSnapshotPath
			}
		}
	}


	if s.snapshotsPath == "" {
		baseSnapshot = nil
	}

	backup := uback.Backup{Snapshot: uback.Snapshot(snapshot), BaseSnapshot: baseSnapshot}
	if tmpSnapshotPath != finalSnapshotPath {
		err := uback.RunCommand(btrfsLog, uback.BuildCommand(s.snapshotCommand, "-r", s.basePath, tmpSnapshotPath))
		if err != nil {
			return uback.Backup{}, nil, err
		}
		btrfsLog.Printf("creating backup: %s", backup.Filename())
	} else {
		btrfsLog.Printf("reusing backup: %s", backup.Filename())
	}

	args := []string{}
	if baseSnapshot != nil {
		args = append(args, "-p", path.Join(s.snapshotsPath, baseSnapshot.Name()))
	}
	args = append(args, tmpSnapshotPath)
	return uback.WrapSourceCommand(backup, uback.BuildCommand(s.sendCommand, args...), func(err error) error {
		if err != nil || s.snapshotsPath == "" {
			_ = uback.RunCommand(btrfsLog, uback.BuildCommand(s.deleteCommand, tmpSnapshotPath))
			return err
		}
		if tmpSnapshotPath != finalSnapshotPath {
			return os.Rename(tmpSnapshotPath, finalSnapshotPath)
		}
		return nil
	})
}

// Part of uback.Source interface
func (s *btrfsSource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	cmd := uback.BuildCommand(s.receiveCommand, targetDir)
	cmd.Stdin = data
	err := uback.RunCommand(btrfsLog, cmd)
	if err != nil {
		return err
	}

	return os.Rename(path.Join(targetDir, "_tmp-"+backup.Snapshot.Name()), path.Join(targetDir, backup.Snapshot.Name()))
}
