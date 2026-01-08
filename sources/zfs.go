package sources

import (
	"bytes"
	"encoding/json"

	uback "github.com/sloonz/uback/lib"

	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrZfsDataset = errors.New("zfs source: invalid dataset")
	zfsLog        = logrus.WithFields(logrus.Fields{
		"source": "zfs",
	})
)

type zfsSource struct {
	options         *uback.Options
	dataset         string
	prefix          string
	replicate       bool
	useBookmarks    bool
	listCommand     []string
	snapshotCommand []string
	bookmarkCommand []string
	sendCommand     []string
	receiveCommand  []string
	destroyCommand  []string
	reuseSnapshots  int
}

type zfsListResult struct {
	Datasets map[string]any `json:"datasets"`
}

func newZfsSource(options *uback.Options) (uback.Source, error) {
	dataset := options.String["Dataset"]
	if dataset == "" {
		return nil, ErrZfsDataset
	}

	var reuseSnapshots int
	if options.String["ReuseSnapshots"] != "" {
		var err error
		reuseSnapshots, err = uback.ParseInterval(options.String["ReuseSnapshots"])
		if err != nil {
			return nil, err
		}
	}

	replicate, err := options.GetBoolean("Replicate", false)
	if err != nil {
		return nil, err
	}

	useBookmarks, err := options.GetBoolean("UseBookmarks", true)
	if err != nil {
		return nil, err
	}

	if replicate && useBookmarks {
		zfsLog.Fatal("Cannot have both Replicate=true and UseBookmarks=true")
	}

	raw, err := options.GetBoolean("Raw", true)
	if err != nil {
		return nil, err
	}

	snapshotCommand := options.GetCommand("SnapshotCommand", []string{"zfs", "snapshot"})
	sendCommand := options.GetCommand("SendCommand", []string{"zfs", "send"})

	if replicate {
		snapshotCommand = append(snapshotCommand, "-r")
		sendCommand = append(sendCommand, "-R")

		if options.String["Exclude"] != "" {
			sendCommand = append(sendCommand, "-X", options.String["Exclude"])
		}
	}

	if raw {
		sendCommand = append(sendCommand, "--raw")
	}

	return &zfsSource{
		options:         options,
		dataset:         dataset,
		replicate:       replicate,
		useBookmarks:    useBookmarks,
		prefix:          options.GetString("Prefix", "uback-"),
		listCommand:     options.GetCommand("ListCommand", []string{"zfs", "list"}),
		snapshotCommand: snapshotCommand,
		bookmarkCommand: options.GetCommand("BookmarkCommand", []string{"zfs", "bookmark"}),
		sendCommand:     sendCommand,
		destroyCommand:  options.GetCommand("DestroyCommand", []string{"zfs", "destroy"}),
		reuseSnapshots:  reuseSnapshots,
	}, nil
}

func newZfsSourceForRestoration(options *uback.Options) (uback.Source, error) {
	dataset := options.String["Dataset"]
	if dataset == "" {
		return nil, ErrZfsDataset
	}

	return &zfsSource{
		dataset:        dataset,
		receiveCommand: options.GetCommand("ReceiveCommand", []string{"zfs", "receive"}),
	}, nil
}

func (s *zfsSource) list(args ...string) (map[string]interface{}, error) {
	buf := bytes.NewBuffer(nil)
	cmd := uback.BuildCommand(s.listCommand, append([]string{"-j", "-o", "name"}, args...)...)
	cmd.Stdout = buf
	if err := uback.RunCommand(zfsLog, cmd); err != nil {
		return nil, err
	}

	var res zfsListResult
	if err := json.NewDecoder(buf).Decode(&res); err != nil {
		return nil, err
	}

	return res.Datasets, nil
}

func (s *zfsSource) listSnapshots(kind, sigil string) ([]uback.Snapshot, error) {
	datasets, err := s.list("-t", kind, "-d", "1", s.dataset)
	if err != nil {
		return nil, err
	}

	var snapshots []uback.Snapshot

	re := regexp.MustCompile(fmt.Sprintf("^%s$", uback.SnapshotRe))
	for name := range datasets {
		if snapshot, ok := strings.CutPrefix(name, s.dataset+sigil+s.prefix); ok {
			if !re.MatchString(snapshot) {
				zfsLog.WithFields(logrus.Fields{"snapshot": name}).Warnf("invalid snapshot name")
				continue
			}

			snapshots = append(snapshots, uback.Snapshot(snapshot))
		}
	}

	return snapshots, nil
}

// Part of uback.Source interface
func (s *zfsSource) ListArchives() ([]uback.Snapshot, error) {
	return s.listSnapshots("snapshot", "@")
}

// Part of uback.Source interface
func (s *zfsSource) ListBookmarks() ([]uback.Snapshot, error) {
	return s.listSnapshots("bookmark", "#")
}

// Part of uback.Source interface
func (s *zfsSource) RemoveArchive(snapshot uback.Snapshot) error {
	destroyCommand := append([]string{}, s.destroyCommand...)
	if s.replicate {
		destroyCommand = append(destroyCommand, "-r")
	}
	cmd := uback.BuildCommand(destroyCommand, s.dataset+"@"+s.prefix+snapshot.Name())
	return uback.RunCommand(zfsLog, cmd)
}

// Part of uback.Source interface
func (s *zfsSource) RemoveBookmark(snapshot uback.Snapshot) error {
	cmd := uback.BuildCommand(s.destroyCommand, s.dataset+"#"+s.prefix+snapshot.Name())
	return uback.RunCommand(zfsLog, cmd)
}

// Part of uback.Source interface
func (s *zfsSource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	reused := false
	snapshot := time.Now().UTC().Format(uback.SnapshotTimeFormat)

	if s.reuseSnapshots != 0 {
		snapshots, err := uback.SortedListArchives(s)
		if err != nil {
			return uback.Backup{}, nil, err
		}

		if len(snapshots) > 0 {
			t, err := snapshots[0].Time()
			if err != nil {
				return uback.Backup{}, nil, err
			}

			if time.Now().UTC().Sub(t).Seconds() <= float64(s.reuseSnapshots) {
				reused = true
				snapshot = string(snapshots[0])
			}
		}
	}

	backup := uback.Backup{Snapshot: uback.Snapshot(snapshot), BaseSnapshot: baseSnapshot}
	if reused {
		zfsLog.Printf("reusing backup: %s", backup.Filename())
	} else {
		zfsLog.Printf("creating backup: %s", backup.Filename())

		err := uback.RunCommand(zfsLog, uback.BuildCommand(s.snapshotCommand, s.dataset+"@"+s.prefix+snapshot))
		if err != nil {
			return uback.Backup{}, nil, err
		}

		if s.useBookmarks {
			cmd := uback.BuildCommand(s.bookmarkCommand, s.dataset+"@"+s.prefix+snapshot, s.dataset+"#"+s.prefix+snapshot)
			if err := uback.RunCommand(zfsLog, cmd); err != nil {
				return uback.Backup{}, nil, err
			}
		}
	}

	args := []string{}
	if baseSnapshot != nil {
		if s.useBookmarks {
			args = append(args, "-i", s.dataset+"#"+s.prefix+baseSnapshot.Name())
		} else {
			args = append(args, "-i", s.dataset+"@"+s.prefix+baseSnapshot.Name())
		}
	}
	args = append(args, s.dataset+"@"+s.prefix+snapshot)
	return uback.WrapSourceCommand(backup, uback.BuildCommand(s.sendCommand, args...), nil)
}

// Part of uback.Source interface
func (s *zfsSource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	cmd := uback.BuildCommand(s.receiveCommand, s.dataset)
	cmd.Stdin = data
	err := uback.RunCommand(zfsLog, cmd)
	if err != nil {
		return err
	}

	return nil
}
