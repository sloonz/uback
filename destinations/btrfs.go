package destinations

import (
	"github.com/sloonz/uback/container"
	"github.com/sloonz/uback/lib"

	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	ErrBtrfsPath = errors.New("btrfs destination: missing path")
	btrfsLog     = logrus.WithFields(logrus.Fields{
		"destination": "btrfs",
	})
)

type btrfsDestination struct {
	options         *uback.Options
	basePath        string
	snapshotCommand []string
	sendCommand     []string
	receiveCommand  []string
	deleteCommand   []string
}

func newBtrfsDestination(options *uback.Options) (uback.Destination, error) {
	basePath := options.String["Path"]
	if basePath == "" {
		return nil, ErrFSPath
	}

	err := os.MkdirAll(basePath, 0777)
	if err != nil {
		return nil, err
	}

	return &btrfsDestination{
		options:         options,
		basePath:        basePath,
		snapshotCommand: options.GetCommand("SnapshotCommand", []string{"btrfs", "subvolume", "snapshot"}),
		sendCommand:     options.GetCommand("SendCommand", []string{"btrfs", "send"}),
		receiveCommand:  options.GetCommand("ReceiveCommand", []string{"btrfs", "receive"}),
		deleteCommand:   options.GetCommand("DeleteCommand", []string{"btrfs", "subvolume", "delete"}),
	}, nil
}

func (d *btrfsDestination) ListBackups() ([]uback.Backup, error) {
	var res []uback.Backup
	entries, err := os.ReadDir(d.basePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") || !entry.IsDir() {
			continue
		}

		backup, err := uback.ParseBackupFilename(entry.Name()+"-full.ubkp", true)
		if err != nil {
			btrfsLog.WithFields(logrus.Fields{
				"file": entry.Name(),
			})
			logrus.Warnf("invalid backup file: %v", err)
			continue
		}

		res = append(res, backup)
	}

	return res, nil
}

func (d *btrfsDestination) RemoveBackup(backup uback.Backup) error {
	return uback.RunCommand(btrfsLog, uback.BuildCommand(d.deleteCommand, path.Join(d.basePath, string(backup.Snapshot.Name()))))
}

func (d *btrfsDestination) SendBackup(backup uback.Backup, data io.Reader) error {
	cr, err := container.NewReader(data)
	if err != nil {
		return err
	}

	err = cr.Unseal(nil)
	if err != nil {
		return err
	}

	cmd := uback.BuildCommand(d.receiveCommand, d.basePath)
	cmd.Stdin = cr
	err = uback.RunCommand(btrfsLog, cmd)
	if err != nil {
		return err
	}

	return os.Rename(path.Join(d.basePath, "_tmp-"+backup.Snapshot.Name()), path.Join(d.basePath, backup.Snapshot.Name()))
}

func (d *btrfsDestination) ReceiveBackup(backup uback.Backup) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		// btrfs source expects snapshots to be named _tmp-(snapshot), create a temporary snapshot
		// matching that name
		cmd := uback.BuildCommand(d.snapshotCommand, "-r", path.Join(d.basePath, backup.Snapshot.Name()), path.Join(d.basePath, "_tmp-"+backup.Snapshot.Name()))
		err := uback.RunCommand(btrfsLog, cmd)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		defer func() {
			cmd := uback.BuildCommand(d.deleteCommand, path.Join(d.basePath, "_tmp-"+backup.Snapshot.Name()))
			_ = uback.RunCommand(btrfsLog, cmd)
		}()

		cw, err := container.NewWriter(pw, nil, "btrfs", 3)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		cmd = uback.BuildCommand(d.sendCommand, path.Join(d.basePath, "_tmp-"+backup.Snapshot.Name()))
		cmd.Stdout = cw
		err = uback.StartCommand(btrfsLog, cmd)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		err = cmd.Wait()

		if err != nil {
			err = cw.Close()
		} else {
			_ = cw.Close()
		}

		pw.CloseWithError(err)
	}()

	return pr, nil
}
