package sources

import (
	_ "embed" // required for go:embed
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	uback "github.com/sloonz/uback/lib"
)

var (
	ErrMariaBackupCommand  = errors.New("mariabackup source: missing or invalid mariabackup command")
	ErrMariadbCommand      = errors.New("mariabackup source: missing or invalid mariadb command")
	ErrParseMariadbVersion = errors.New("mariabackup source: cannot parse version information")
	mariaBackupLog         = logrus.WithFields(logrus.Fields{
		"source": "mariabackup",
	})

	//go:embed scripts/sqldump-podman.sh
	mariaBackupPodmanScript []byte

	//go:embed scripts/sqldump-local.sh
	mariaBackupLocalScript []byte
)

func readBackupInfo(basePath string) ([]byte, error) {
	info, err := os.ReadFile(path.Join(basePath, "xtrabackup_info"))
	if err == nil {
		return info, nil
	}

	if os.IsNotExist(err) {
		info, err = os.ReadFile(path.Join(basePath, "mariadb_backup_info"))
		if err == nil {
			return info, nil
		}
	}

	return nil, err
}

type mariaBackupSource struct {
	options           *uback.Options
	snapshotsPath     string
	command           []string
	mdbVersionCommand []string
	authFileData      string
	versionCheck      bool
	usePodman         bool
}

func newMariaBackupSource(options *uback.Options) (uback.Source, error) {
	snapshotsPath := options.String["SnapshotsPath"]
	if snapshotsPath == "" {
		mariaBackupLog.Warnf("SnapshotsPath option missing, incremental backups will be impossible")
	} else {
		err := os.MkdirAll(snapshotsPath, 0777)
		if err != nil {
			return nil, err
		}
	}

	versionCheck, err := options.GetBoolean("VersionCheck", true)
	if err != nil {
		return nil, err
	}

	command := options.GetCommand("Command", []string{"mariadb-backup"})
	if len(command) == 0 {
		return nil, ErrMariaBackupCommand
	}

	mdbVersionCommand := options.GetCommand("MariadbCommand", []string{"mariadb"})
	if len(mdbVersionCommand) == 0 && versionCheck {
		return nil, ErrMariadbCommand
	}

	authFileData := ""
	user, hasUser := options.String["User"]
	pwd, hasPassword := options.String["Password"]
	if hasUser || hasPassword {
		authFileData = "[mariabackup]\n"
		if hasUser {
			authFileData += fmt.Sprintf("user=%s\n", user)
		}
		if hasPassword {
			authFileData += fmt.Sprintf("password=%s\n", pwd)
		}
	}

	return &mariaBackupSource{
		options:           options,
		snapshotsPath:     snapshotsPath,
		command:           command,
		mdbVersionCommand: mdbVersionCommand,
		versionCheck:      versionCheck,
		authFileData:      authFileData}, nil
}

func newMariaBackupSourceForRestoration(options *uback.Options) (uback.Source, error) {
	command := options.GetCommand("Command", []string{"mariadb-backup"})
	if len(command) == 0 {
		return nil, ErrMariaBackupCommand
	}

	usePodman, err := options.GetBoolean("UsePodman", true)
	if err != nil {
		return nil, err
	}

	return &mariaBackupSource{command: command, usePodman: usePodman}, nil
}

// Part of uback.Source interface
func (s *mariaBackupSource) ListArchives() ([]uback.Snapshot, error) {
	return nil, nil
}

// Part of uback.Source interface
func (s *mariaBackupSource) ListBookmarks() ([]uback.Snapshot, error) {
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
			mariaBackupLog.WithFields(logrus.Fields{"file": entry.Name()}).Warnf("invalid snapshot name")
			continue
		}

		snapshots = append(snapshots, uback.Snapshot(entry.Name()))
	}

	return snapshots, nil
}

// Part of uback.Source interface
func (s *mariaBackupSource) RemoveArchive(snapshot uback.Snapshot) error {
	panic("should never happen")
}

// Part of uback.Source interface
func (s *mariaBackupSource) RemoveBookmark(snapshot uback.Snapshot) error {
	if s.snapshotsPath == "" {
		return nil
	}
	return os.RemoveAll(path.Join(s.snapshotsPath, string(snapshot)))
}

// Part of uback.Source interface
func (s *mariaBackupSource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	snapshot := time.Now().UTC().Format(uback.SnapshotTimeFormat)
	tmpSnapshotPath := path.Join(s.snapshotsPath, fmt.Sprintf("_tmp-%s", snapshot))
	finalSnapshotPath := path.Join(s.snapshotsPath, snapshot)

	command := make([]string, 0, len(s.command)+5)
	command = append(command, s.command...)

	var authFile *os.File
	var err error
	if s.authFileData != "" {
		authFile, err = os.CreateTemp(s.snapshotsPath, "")
		if err != nil {
			return uback.Backup{}, nil, err
		}
		_, err = authFile.Write([]byte(s.authFileData))
		if err != nil {
			return uback.Backup{}, nil, err
		}
		err = authFile.Sync()
		if err != nil {
			return uback.Backup{}, nil, err
		}
		command = append(command, fmt.Sprintf("--defaults-file=%s", authFile.Name()))
	}

	var serverVersion []byte
	mdbVersionCommand := append([]string{}, s.mdbVersionCommand...)
	if s.authFileData != "" {
		mdbVersionCommand = append(mdbVersionCommand, fmt.Sprintf("--defaults-file=%s", authFile.Name()))
	}
	mdbVersionCommand = append(mdbVersionCommand, "-BNe", "select version()")

	if baseSnapshot != nil && s.versionCheck {
		baseInfo, err := readBackupInfo(path.Join(s.snapshotsPath, baseSnapshot.Name()))
		if err != nil {
			return uback.Backup{}, nil, err
		}

		snapshotVersionMatch := regexp.MustCompile(`(?m)^server_version\s*=\s*(\d+\.\d+\.\d+)`).FindSubmatch(baseInfo)
		if len(snapshotVersionMatch) != 2 {
			return uback.Backup{}, nil, ErrParseMariadbVersion
		}

		cmd := uback.BuildCommand(mdbVersionCommand)
		cmd.Stdout = nil
		serverVersion, err = cmd.Output()
		if err != nil {
			return uback.Backup{}, nil, fmt.Errorf("cannot get mariadb server version: %v", err)
		}

		serverVersionMatch := regexp.MustCompile(`\d+\.\d+\.\d+`).Find(serverVersion)
		if string(serverVersionMatch) != string(snapshotVersionMatch[1]) {
			logrus.Warnf("mismatch between base backup server version (%s) and current server version (%s), forcing incremental backup", string(snapshotVersionMatch[1]), string(serverVersionMatch))
			baseSnapshot = nil
		}
	}

	command = append(command, "--backup")
	command = append(command, "--stream=mbstream")
	if s.snapshotsPath != "" {
		command = append(command, fmt.Sprintf("--extra-lsndir=%s", tmpSnapshotPath))
		if baseSnapshot != nil {
			command = append(command, fmt.Sprintf("--incremental-basedir=%s", path.Join(s.snapshotsPath, baseSnapshot.Name())))
		}
	} else {
		baseSnapshot = nil
	}

	backup := uback.Backup{Snapshot: uback.Snapshot(snapshot), BaseSnapshot: baseSnapshot}
	mariaBackupLog.Printf("creating backup: %s", backup.Filename())

	return uback.WrapSourceCommand(backup, uback.BuildCommand(command), func(err error) error {
		if serverVersion != nil {
			cmd := uback.BuildCommand(mdbVersionCommand)
			cmd.Stdout = nil
			newServerVersion, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("cannot get mariadb server version: %v", err)
			}

			if string(serverVersion) != string(newServerVersion) {
				return errors.New("race condition: server changed its version during backup")
			}
		}
		if authFile != nil {
			n := authFile.Name()
			authFile.Close()
			os.Remove(n)
		}
		if err != nil {
			if s.snapshotsPath != "" {
				os.RemoveAll(tmpSnapshotPath)
			}
			return err
		}
		if s.snapshotsPath != "" {
			return os.Rename(tmpSnapshotPath, finalSnapshotPath)
		}
		return nil

	})
}

// Part of uback.Source interface
func (s *mariaBackupSource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	err := os.RemoveAll(path.Join(targetDir, backup.Snapshot.Name()))
	if err != nil {
		return err
	}

	restoreDir := path.Join(targetDir, backup.Snapshot.Name())
	err = os.MkdirAll(restoreDir, 0777)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(restoreDir, "sqldump-podman.sh"), mariaBackupPodmanScript, 0777)
	if err != nil {
		mariaBackupLog.Warnf("cannot write sqldump-podman.sh script: %v", err)
	}

	err = os.WriteFile(path.Join(restoreDir, "sqldump-local.sh"), mariaBackupLocalScript, 0777)
	if err != nil {
		mariaBackupLog.Warnf("cannot write sqldump-local.sh script: %v", err)
	}

	var extractCommand []string
	if s.usePodman {
		extractCommand = append(extractCommand, "podman", "run", "--rm", "-v", fmt.Sprintf("%s:%s", targetDir, targetDir), "-i", "docker.io/library/mariadb:latest")
	}

	cmd := uback.BuildCommand(extractCommand, "mbstream", "-x", "-C", restoreDir)
	cmd.Stdin = data
	err = uback.RunCommand(mariaBackupLog, cmd)
	if err != nil {
		return err
	}

	var prepareCommand []string
	if s.usePodman {
		info, err := readBackupInfo(restoreDir)
		if err != nil {
			return err
		}

		versionMatch := regexp.MustCompile(`(?m)^server_version\s*=\s*(\d+\.\d+\.\d+)`).FindSubmatch(info)
		if len(versionMatch) != 2 {
			return err
		}

		version := string(versionMatch[1])
		prepareCommand = append(prepareCommand, "podman", "run", "--rm", "-v", fmt.Sprintf("%v:%v", targetDir, targetDir), "-i", fmt.Sprintf("docker.io/library/mariadb:%s", version), "mariadb-backup")
	} else {
		prepareCommand = append(prepareCommand, s.command...)
	}

	if backup.BaseSnapshot != nil {
		baseDir := path.Join(targetDir, backup.BaseSnapshot.Name())
		cmd = uback.BuildCommand(prepareCommand, "--prepare", fmt.Sprintf("--target-dir=%s", baseDir), fmt.Sprintf("--incremental-dir=%s", restoreDir))
		err = uback.RunCommand(mariaBackupLog, cmd)
		if err != nil {
			return err
		}

		err = os.RemoveAll(restoreDir)
		if err != nil {
			return err
		}

		return os.Rename(baseDir, restoreDir)
	}

	cmd = uback.BuildCommand(prepareCommand, "--prepare", fmt.Sprintf("--target-dir=%s", restoreDir))
	return uback.RunCommand(mariaBackupLog, cmd)
}
