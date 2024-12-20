package destinations

import (
	uback "github.com/sloonz/uback/lib"

	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/secsy/goftp"
	"github.com/sirupsen/logrus"
)

var (
	ftpLog = logrus.WithFields(logrus.Fields{
		"destination": "ftp",
	})
)

type ftpDestination struct {
	options *uback.Options
	prefix  string
	client  *goftp.Client
}

func newFTPDestination(options *uback.Options) (uback.Destination, error) {
	u, err := url.Parse(options.String["URL"])
	if err != nil {
		ftpLog.Warnf("cannot parse URL: %v", err)
		return nil, fmt.Errorf("invalid FTP URL: %v", err)
	}

	address := u.Host
	username := u.User.Username()
	password, _ := u.User.Password()
	prefix := strings.Trim(options.String["Prefix"], "/") + "/"
	if prefix == "/" {
		prefix = ""
	}

	config := goftp.Config{
		User:     username,
		Password: password,
	}

	client, err := goftp.DialConfig(config, address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP server: %v", err)
	}

	return &ftpDestination{options: options, client: client, prefix: prefix}, nil
}

func (d *ftpDestination) makePrefix() error {
	var err error

	if d.prefix == "" {
		return nil
	}

	dirs := strings.Split(strings.Trim(d.prefix, "/"), "/")
	currentPath := ""

	for _, dir := range dirs {
		currentPath = path.Join(currentPath, dir)
		_, err = d.client.Mkdir(currentPath)
	}

	return err
}

func (d *ftpDestination) ListBackups() ([]uback.Backup, error) {
	var res []uback.Backup

	_ = d.makePrefix()
	files, err := d.client.ReadDir(d.prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups on FTP server: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), ".") || strings.HasPrefix(file.Name(), "_") {
			continue
		}

		backup, err := uback.ParseBackupFilename(file.Name(), true)
		if err != nil {
			ftpLog.WithFields(logrus.Fields{
				"key": file.Name(),
			}).Warnf("invalid backup file: %v", err)
			continue
		}

		res = append(res, backup)
	}

	return res, nil
}

func (d *ftpDestination) RemoveBackup(backup uback.Backup) error {
	filePath := path.Join(d.prefix, backup.Filename())
	if err := d.client.Delete(filePath); err != nil {
		return fmt.Errorf("failed to remove backup from FTP server: %v", err)
	}
	return nil
}

func (d *ftpDestination) SendBackup(backup uback.Backup, data io.Reader) error {
	tmpFilePath := path.Join(d.prefix, "_tmp"+backup.Filename())
	finalFilePath := path.Join(d.prefix, backup.Filename())
	ftpLog.Printf("writing backup to temporary file %s", tmpFilePath)

	_ = d.makePrefix()
	if err := d.client.Store(tmpFilePath, data); err != nil {
		return fmt.Errorf("failed to write temporary backup file to FTP server: %v", err)
	}

	ftpLog.Printf("renaming temporary file %s to %s", tmpFilePath, finalFilePath)
	if err := d.client.Rename(tmpFilePath, finalFilePath); err != nil {
		_ = d.client.Delete(tmpFilePath)
		return fmt.Errorf("failed to rename temporary backup file on FTP server: %v", err)
	}

	return nil
}

func (d *ftpDestination) ReceiveBackup(backup uback.Backup) (io.ReadCloser, error) {
	filePath := path.Join(d.prefix, backup.Filename())

	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		if err := d.client.Retrieve(filePath, writer); err != nil {
			writer.CloseWithError(fmt.Errorf("failed to read backup from FTP server: %v", err))
		}
	}()

	return reader, nil
}
