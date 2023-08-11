package cmd

import (
	"github.com/sloonz/uback/lib"

	"io"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cmdFetchRecursive bool
	cmdFetchTargetDir string
	cmdFetch          = &cobra.Command{
		Use:   "fetch <destination> [backup-name]",
		Short: "Fetch a backup file (default: last backup) from a destination",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			targetName := ""
			if len(args) > 1 {
				targetName = args[1]
			}

			dstOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
				WithDestination().
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

			fetchedBackups := []uback.Backup{*targetBackup}
			if cmdFetchRecursive {
				var ok bool
				fetchedBackups, ok = uback.GetFullChain(*targetBackup, uback.MakeIndex(backups))
				if !ok {
					logrus.Warn("the incremental backups chain do not reference a final full backup")
				}
			}

			for i := len(fetchedBackups) - 1; i >= 0; i-- {
				b := fetchedBackups[i]
				logrus.Printf("fetching %v", b.Filename())
				data, err := dstOpts.Destination.ReceiveBackup(b)
				if err != nil {
					logrus.Fatal(err)
				}
				defer data.Close()

				f, err := os.OpenFile(path.Join(cmdFetchTargetDir, b.Filename()), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
				if err != nil {
					logrus.Fatal(err)
				}
				defer f.Close()

				_, err = io.Copy(f, data)
				if err != nil {
					logrus.Fatal(err)
				}
			}
		},
	}
)

func init() {
	cmdFetch.Flags().BoolVarP(&cmdFetchRecursive, "recursive", "r", false, "fetch dependencies of incremental backups")
	cmdFetch.Flags().StringVarP(&cmdFetchTargetDir, "target-dir", "d", ".", "target dir")
}
