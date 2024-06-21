package cmd

import (
	"github.com/sloonz/uback/container"
	"github.com/sloonz/uback/lib"
	"github.com/sloonz/uback/sources"

	"io"
	"os"
	"path"
	"strings"

	"filippo.io/age"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func restore(dst uback.Destination, b uback.Backup, sk []age.Identity, targetDir string) error {
	logrus.Printf("restoring %v onto %v", b.Filename(), targetDir)

	srcOpts, err := uback.EvalOptions(uback.SplitOptions(cmdRestoreSourceOptions), presets)
	if err != nil {
		logrus.Fatal(err)
	}

	var data io.ReadCloser
	if f, err := os.Open(path.Join(targetDir, b.Filename())); err == nil {
		data = f
	} else {
		data, err = dst.ReceiveBackup(b)
		if err != nil {
			return err
		}
	}
	defer data.Close()

	r, err := container.NewReader(data)
	if err != nil {
		return err
	}
	defer r.Close()

	src, err := sources.NewForRestoration(srcOpts, r.Options.String["Type"])
	if err != nil {
		return err
	}

	err = r.Unseal(sk)
	if err != nil {
		return err
	}

	err = src.RestoreBackup(targetDir, b, r)
	if err != nil {
		return err
	}

	return nil
}

var (
	cmdRestoreTargetDir     string
	cmdRestoreSourceOptions string
	cmdRestoreUseLocal      bool
	cmdRestore              = &cobra.Command{
		Use:   "restore <dest> [backup-name]",
		Short: "Restore a backup",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			targetName := ""
			if len(args) > 1 {
				targetName = args[1]
			}

			dstOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
				WithDestination().
				WithIdentities().
				FatalOnError()

			backups, err := uback.SortedListBackups(dstOpts.Destination)
			if err != nil {
				logrus.Fatal(err)
			}

			var targetBackup *uback.Backup
			for i, b := range backups {
				if strings.HasPrefix(b.FullName(), targetName) {
					targetBackup = &backups[i]
					break
				}
			}
			if targetBackup == nil {
				logrus.Fatal("cannot find backup")
			}

			fetchedBackups, ok := uback.GetFullChain(*targetBackup, uback.MakeIndex(backups))
			if !ok {
				logrus.Fatal("the incremental backups chain do not reference a final full backup")
			}

			for i := len(fetchedBackups) - 1; i >= 0; i-- {
				err = restore(dstOpts.Destination, fetchedBackups[i], dstOpts.Identities, cmdRestoreTargetDir)
				if err != nil {
					logrus.Fatal(err)
				}
			}
		},
	}
)

func init() {
	cmdRestore.Flags().StringVarP(&cmdRestoreTargetDir, "target-dir", "d", ".", "target dir")
	cmdRestore.Flags().StringVarP(&cmdRestoreSourceOptions, "source-options", "o", ".", "additional source options")
	cmdRestore.Flags().BoolVarP(&cmdRestoreUseLocal, "local", "l", false, "use local backup files if present")
}
