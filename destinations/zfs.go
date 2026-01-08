package destinations

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/sloonz/uback/container"
	uback "github.com/sloonz/uback/lib"

	"errors"
	"io"

	"github.com/sirupsen/logrus"
)

var (
	ErrZfsDataset = errors.New("zfs destination: missing dataset")
	zfsLog        = logrus.WithFields(logrus.Fields{
		"destination": "zfs",
	})
)

type zfsDestination struct {
	options        *uback.Options
	dataset        string
	prefix         string
	listCommand    []string
	sendCommand    []string
	receiveCommand []string
	destroyCommand []string
}

func newZfsDestination(options *uback.Options) (uback.Destination, error) {
	dataset := options.String["Dataset"]
	if dataset == "" {
		return nil, ErrZfsDataset
	}

	replicate, err := options.GetBoolean("Replicate", true)
	if err != nil {
		return nil, err
	}

	raw, err := options.GetBoolean("Raw", true)
	if err != nil {
		return nil, err
	}

	sendCommand := options.GetCommand("SendCommand", []string{"zfs", "send"})
	destroyCommand := options.GetCommand("DestroyCommand", []string{"zfs", "detroy"})

	if replicate {
		sendCommand = append(sendCommand, "-R")
		destroyCommand = append(destroyCommand, "-r")

		if options.String["Exclude"] != "" {
			sendCommand = append(sendCommand, "-X", options.String["Exclude"])
		}
	}

	if raw {
		sendCommand = append(sendCommand, "--raw")
	}

	return &zfsDestination{
		options:        options,
		dataset:        dataset,
		prefix:         options.GetString("Prefix", "uback-"),
		listCommand:    options.GetCommand("ListCommand", []string{"zfs", "list"}),
		sendCommand:    sendCommand,
		receiveCommand: options.GetCommand("ReceiveCommand", []string{"zfs", "receive"}),
		destroyCommand: destroyCommand,
	}, nil
}

func (d *zfsDestination) ListBackups() ([]uback.Backup, error) {
	buf := bytes.NewBuffer(nil)
	cmd := uback.BuildCommand(d.listCommand, "-j", "-d", "1", "-o", "name", "-t", "snapshot", d.dataset)
	cmd.Stdout = buf
	if err := uback.RunCommand(zfsLog, cmd); err != nil {
		// Assume that command failed because dataset does not exist yet
		logrus.Warnf("Listing of ZFS snapshots failed: %v. Dataset does not exists yet ?", err)
		return nil, nil
	}

	var res struct {
		Datasets map[string]any `json:"datasets"`
	}
	if err := json.NewDecoder(buf).Decode(&res); err != nil {
		return nil, err
	}

	var backups []uback.Backup
	for name := range res.Datasets {
		if snapshot, ok := strings.CutPrefix(name, d.dataset+"@"+d.prefix); ok {
			backups = append(backups, uback.Backup{Snapshot: uback.Snapshot(snapshot)})
		}
	}

	return backups, nil
}

func (d *zfsDestination) RemoveBackup(backup uback.Backup) error {
	return uback.RunCommand(zfsLog, uback.BuildCommand(d.destroyCommand, d.dataset+"@"+d.prefix+backup.BaseSnapshot.Name()))
}

func (d *zfsDestination) SendBackup(backup uback.Backup, data io.Reader) error {
	cr, err := container.NewReader(data)
	if err != nil {
		return err
	}

	err = cr.Unseal(nil)
	if err != nil {
		return err
	}

	cmd := uback.BuildCommand(d.receiveCommand, "-o", "readonly=on", d.dataset)
	cmd.Stdin = cr
	return uback.RunCommand(zfsLog, cmd)
}

func (d *zfsDestination) ReceiveBackup(backup uback.Backup) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	cmd := uback.BuildCommand(d.sendCommand, d.dataset+"@"+d.prefix+backup.Snapshot.Name())
	cmd.Stdout = pw
	err := uback.StartCommand(commandLog, cmd)
	if err != nil {
		return nil, err
	}

	go func() {
		pw.CloseWithError(cmd.Wait())
	}()

	return pr, nil
}
