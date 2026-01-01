package cmd

import (
	uback "github.com/sloonz/uback/lib"

	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cmdListBackups = &cobra.Command{
	Use:   "backups <destination>",
	Short: "List backups on a destination",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dstOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
			WithDestination().
			FatalOnError()

		backups, err := uback.SortedListBackups(dstOpts.Destination)
		if err != nil {
			logrus.Fatal(err)
		}

		for i := len(backups) - 1; i >= 0; i-- {
			b := backups[i]
			if b.BaseSnapshot == nil {
				fmt.Printf("%s (full)\n", b.Snapshot.Name())
			} else {
				fmt.Printf("%s (base: %s)\n", b.Snapshot.Name(), b.BaseSnapshot.Name())
			}
		}
	},
}

var cmdListArchives = &cobra.Command{
	Use:   "archives <source>",
	Short: "List archives on a source",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		srcOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
			WithSource().
			FatalOnError()

		snapshots, err := uback.SortedListArchives(srcOpts.Source)
		if err != nil {
			logrus.Fatal(err)
		}

		for i := len(snapshots) - 1; i >= 0; i-- {
			s := snapshots[i]
			fmt.Println(string(s))
		}
	},
}

var cmdListBookmarks = &cobra.Command{
	Use:   "bookmarks <source>",
	Short: "List bookmarks on a source",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		srcOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
			WithSource().
			FatalOnError()

		snapshots, err := uback.SortedListBookmarks(srcOpts.Source)
		if err != nil {
			logrus.Fatal(err)
		}

		for i := len(snapshots) - 1; i >= 0; i-- {
			s := snapshots[i]
			fmt.Println(string(s))
		}
	},
}

var cmdList = &cobra.Command{
	Use: "list",
}

func init() {
	cmdList.AddCommand(cmdListArchives, cmdListBookmarks, cmdListBackups)
}
