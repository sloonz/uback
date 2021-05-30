package destinations

import (
	"github.com/sloonz/uback/lib"

	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	ErrFSPath = errors.New("fs destination: missing path")
	fsLog     = logrus.WithFields(logrus.Fields{
		"destination": "fs",
	})
)

type fsDestination struct {
	options  *uback.Options
	basePath string
}

func newFSDestination(options *uback.Options) (uback.Destination, error) {
	basePath := options.String["Path"]
	if basePath == "" {
		return nil, ErrFSPath
	}

	err := os.MkdirAll(basePath, 0777)
	if err != nil {
		return nil, err
	}

	return &fsDestination{options: options, basePath: basePath}, nil
}

func (d *fsDestination) ListBackups() ([]uback.Backup, error) {
	var res []uback.Backup
	entries, err := os.ReadDir(d.basePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") || entry.IsDir() {
			continue
		}

		backup, err := uback.ParseBackupFilename(entry.Name(), true)
		if err != nil {
			fsLog.WithFields(logrus.Fields{
				"file": entry.Name(),
			})
			logrus.Warnf("invalid backup file: %v", err)
			continue
		}

		res = append(res, backup)
	}

	return res, nil
}

func (d *fsDestination) RemoveBackup(backup uback.Backup) error {
	return os.Remove(path.Join(d.basePath, backup.Filename()))
}

func (d *fsDestination) SendBackup(backup uback.Backup, data io.Reader) error {
	tmpFilename := path.Join(d.basePath, "_tmp-"+backup.Filename())
	finalFilename := path.Join(d.basePath, backup.Filename())
	tmpF, err := os.Create(tmpFilename)
	if err != nil {
		return err
	}
	defer tmpF.Close()
	defer os.Remove(tmpFilename)

	fsLog.Printf("writing backup to %s", tmpFilename)
	_, err = io.Copy(tmpF, data)
	if err != nil {
		return err
	}

	tmpF.Close()

	fsLog.Printf("moving final backup to %s", finalFilename)
	return os.Rename(tmpFilename, finalFilename)
}

func (d *fsDestination) ReceiveBackup(backup uback.Backup) (io.ReadCloser, error) {
	return os.Open(path.Join(d.basePath, backup.Filename()))
}
