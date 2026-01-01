package cmd

import (
	uback "github.com/sloonz/uback/lib"

	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cmdPruneBackupsDryRun bool
var cmdPruneBackups = &cobra.Command{
	Use:   "backups <destination>",
	Short: "Prune backups on a destination",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dstOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
			WithDestination().
			WithRetentionPolicies().
			FatalOnError()

		allBackups, err := uback.SortedListBackups(dstOpts.Destination)
		if err != nil {
			logrus.Fatal(err)
		}

		prunedBackups, err := uback.GetPrunedBackups(allBackups, dstOpts.RetentionPolicies)
		if err != nil {
			logrus.Fatal(err)
		}

		for _, b := range prunedBackups {
			fmt.Println(string(b.Snapshot))
			if !cmdPruneBackupsDryRun {
				err = dstOpts.Destination.RemoveBackup(b)
				if err != nil {
					logrus.WithFields(logrus.Fields{"backup": string(b.Snapshot)}).Warnf("cannot remove backup: %v", err)
				}
			}
		}
	},
}

var cmdPruneSnapshotsDryRun bool
var cmdPruneSnapshots = &cobra.Command{
	Use:   "snapshots <source>",
	Short: "Prune snapshots on a source",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		srcOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
			WithSource().
			WithRetentionPolicies().
			WithStateFile().
			FatalOnError()

		archives, err := uback.SortedListArchives(srcOpts.Source)
		if err != nil {
			logrus.Fatal(err)
		}

		bookmarks, err := uback.SortedListBookmarks(srcOpts.Source)
		if err != nil {
			logrus.Fatal(err)
		}

		state := make(map[string]string)
		if srcOpts.Options.String["StateFile"] != "" {
			rawState, err := os.ReadFile(srcOpts.Options.String["StateFile"])
			if err != nil && !os.IsNotExist(err) {
				logrus.Fatal(err)
			} else if err != nil {
				logrus.Warn("state file does not exists yet ; this is probably a configuration mistake, forcing --dry-run")
				cmdPruneSnapshotsDryRun = true
			}

			if rawState != nil {
				err = json.Unmarshal(rawState, &state)
				if err != nil {
					logrus.Fatal(err)
				}
			}
		}

		prunedArchives, prunedBookmarks, err := uback.GetPrunedSnapshots(archives, bookmarks, srcOpts.RetentionPolicies, state)
		if err != nil {
			logrus.Fatal(err)
		}

		for _, s := range prunedArchives {
			fmt.Println(string(s))
			if !cmdPruneSnapshotsDryRun {
				err = srcOpts.Source.RemoveArchive(s)
				if err != nil {
					logrus.WithFields(logrus.Fields{"archive": string(s)}).Warnf("cannot remove archive: %v", err)
				}
			}
		}

		for _, s := range prunedBookmarks {
			fmt.Println(string(s))
			if !cmdPruneSnapshotsDryRun {
				err = srcOpts.Source.RemoveBookmark(s)
				if err != nil {
					logrus.WithFields(logrus.Fields{"bookmark": string(s)}).Warnf("cannot remove bookmark: %v", err)
				}
			}
		}
	},
}

var cmdPrune = &cobra.Command{
	Use: "prune",
}

func init() {
	cmdPruneBackups.Flags().BoolVarP(&cmdPruneBackupsDryRun, "dry-run", "n", false, "do not actually remove anything, just prints backups that would be removed")
	cmdPruneSnapshots.Flags().BoolVarP(&cmdPruneSnapshotsDryRun, "dry-run", "n", false, "do not actually remove anything, just prints snapshots that would be removed")
	cmdPrune.AddCommand(cmdPruneSnapshots, cmdPruneBackups)
}
