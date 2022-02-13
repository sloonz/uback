package sources

import (
	"github.com/sloonz/uback/lib"

	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/sirupsen/logrus"
)

var (
	ErrCommandMissing = errors.New("command source: invalid path")
	commandLog        = logrus.WithFields(logrus.Fields{
		"source": "command",
	})
)

type commandSource struct {
	options *uback.Options
	command []string
	env     []string
}

func newCommandSource(options *uback.Options) (uback.Source, string, error) {
	command := options.GetCommand("Command", nil)
	if len(command) == 0 {
		return nil, "", ErrCommandMissing
	}

	env := os.Environ()
	for k, v := range options.String {
		env = append(env, fmt.Sprintf("UBACK_OPT_%s=%s", flect.New(k).Underscore().ToUpper().String(), v))
	}
	for k, v := range options.StrSlice {
		jsonVal, err := json.Marshal(v)
		if err != nil {
			return nil, "", err
		}
		env = append(env, fmt.Sprintf("UBACK_SOPT_%s=%s", flect.New(k).Underscore().ToUpper().String(), string(jsonVal)))
	}

	buf := bytes.NewBuffer(nil)
	cmd := uback.BuildCommand(command, "source", "type")
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	cmd.Env = env
	err := cmd.Run()
	if err != nil {
		return nil, "", err
	}

	typ := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(typ, "command:") {
		if strings.HasPrefix(typ, ":") {
			typ = typ[1:]
		} else {
			commandLog.Warn("missing 'command:' prefix")
			typ = "command:" + typ
		}
	}

	return &commandSource{options: options, command: command, env: env}, typ, nil
}

func newCommandSourceForRestoration(command []string, options *uback.Options) (uback.Source, error) {
	return &commandSource{command: command, options: options}, nil
}

// Part of uback.Source interface
func (s *commandSource) ListSnapshots() ([]uback.Snapshot, error) {
	buf := bytes.NewBuffer(nil)
	cmd := uback.BuildCommand(s.command, "source", "list-snapshots")
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	cmd.Env = s.env
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	var snapshots []uback.Snapshot

	re := regexp.MustCompile(fmt.Sprintf("^%s$", uback.SnapshotRe))
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

		if !re.MatchString(entry) {
			commandLog.WithFields(logrus.Fields{"entry": entry}).Warnf("invalid snapshot name")
			continue
		}

		snapshots = append(snapshots, uback.Snapshot(entry))
	}

	return snapshots, nil
}

// Part of uback.Source interface
func (s *commandSource) RemoveSnapshot(snapshot uback.Snapshot) error {
	cmd := uback.BuildCommand(s.command, "source", "remove-snapshot", snapshot.Name())
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = s.env
	return cmd.Run()
}

// Part of uback.Source interface
func (s *commandSource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	var cmd *exec.Cmd
	if baseSnapshot != nil {
		cmd = uback.BuildCommand(s.command, "source", "create-backup", baseSnapshot.Name())
	} else {
		cmd = uback.BuildCommand(s.command, "source", "create-backup")
	}

	commandLog.Printf("running: %v", cmd.String())

	pr, pw := io.Pipe()

	cmd.Stdout = pw
	cmd.Stderr = os.Stderr
	cmd.Env = s.env
	err := cmd.Start()
	if err != nil {
		return uback.Backup{}, nil, err
	}

	go func() {
		pw.CloseWithError(cmd.Wait())
	}()

	br := bufio.NewReader(pr)
	backupName, err := br.ReadString('\n')
	if err != nil {
		cmd.Process.Kill()
		return uback.Backup{}, nil, err
	}

	backup, err := uback.ParseBackupFilename(strings.TrimSpace(backupName), false)
	if err != nil {
		cmd.Process.Kill()
		return uback.Backup{}, nil, err
	}

	return backup, io.NopCloser(br), nil
}

// Part of uback.Source interface
func (s *commandSource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	var cmd *exec.Cmd
	if backup.BaseSnapshot != nil {
		cmd = uback.BuildCommand(s.command, "source", "restore-backup", targetDir, backup.Snapshot.Name(), backup.BaseSnapshot.Name())
	} else {
		cmd = uback.BuildCommand(s.command, "source", "restore-backup", targetDir, backup.Snapshot.Name())
	}

	commandLog.Printf("running: %v", cmd.String())
	cmd.Stdin = data
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
