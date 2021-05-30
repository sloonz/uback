package destinations

import (
	"github.com/sloonz/uback/lib"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/sirupsen/logrus"
)

var (
	ErrCommandMissing = errors.New("command destination: missing command")
	commandLog        = logrus.WithFields(logrus.Fields{
		"destination": "fs",
	})
)

type commandDestination struct {
	options *uback.Options
	command string
	env     []string
}

func newCommandDestination(options *uback.Options) (uback.Destination, error) {
	command := options.String["Command"]
	if command == "" {
		return nil, ErrCommandMissing
	}

	env := os.Environ()
	for k, v := range options.String {
		env = append(env, fmt.Sprintf("UBACK_OPT_%s=%s", flect.New(k).Underscore().ToUpper().String(), v))
	}
	for k, v := range options.StrSlice {
		jsonVal, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		env = append(env, fmt.Sprintf("UBACK_SOPT_%s=%s", flect.New(k).Underscore().ToUpper().String(), string(jsonVal)))
	}

	buf := bytes.NewBuffer(nil)
	cmd := exec.Command(command, "destination", "validate-options")
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	cmd.Env = env
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	return &commandDestination{options: options, command: command, env: env}, nil
}

func (d *commandDestination) ListBackups() ([]uback.Backup, error) {
	var res []uback.Backup

	buf := bytes.NewBuffer(nil)
	cmd := exec.Command(d.command, "destination", "list-backups")
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	cmd.Env = d.env
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	for {
		entry, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.HasPrefix(entry, ".") || strings.HasPrefix(entry, "_") {
			continue
		}

		backup, err := uback.ParseBackupFilename(entry, false)
		if err != nil {
			commandLog.WithFields(logrus.Fields{
				"entry": entry,
			})
			logrus.Warnf("invalid backup file: %v", err)
			continue
		}

		res = append(res, backup)
	}

	return res, nil
}

func (d *commandDestination) RemoveBackup(backup uback.Backup) error {
	cmd := exec.Command(d.command, "destination", "remove-backup", backup.FullName())
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = d.env
	return cmd.Run()
}

func (d *commandDestination) SendBackup(backup uback.Backup, data io.Reader) error {
	cmd := exec.Command(d.command, "destination", "send-backup", backup.FullName())
	cmd.Stdin = data
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = d.env
	return cmd.Run()
}

func (d *commandDestination) ReceiveBackup(backup uback.Backup) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	cmd := exec.Command(d.command, "destination", "receive-backup", backup.FullName())
	cmd.Stdout = pw
	cmd.Stderr = os.Stderr
	cmd.Env = d.env

	commandLog.Printf("running: %v", cmd.String())
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		pw.CloseWithError(cmd.Wait())
	}()

	return pr, nil
}
