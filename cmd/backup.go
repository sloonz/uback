package cmd

import (
	"github.com/sloonz/uback/container"
	"github.com/sloonz/uback/lib"

	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cmdBackupForceFull bool
	cmdBackupNoPrune   bool

	cmdBackup = &cobra.Command{
		Use:   "backup <source> <destination>",
		Short: "Create a backup",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			srcOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[0]), presets)).
				WithSource().
				WithRetentionPolicies().
				WithPublicKey().
				WithStateFile().
				FatalOnError()

			dstOpts := newOptionsBuilder(uback.EvalOptions(uback.SplitOptions(args[1]), presets)).
				WithDestination().
				WithStringOption("ID").
				WithRetentionPolicies().
				FatalOnError()

			compressionLevel := 3
			// TODO: read compression level from options

			backups, err := uback.SortedListBackups(dstOpts.Destination)
			if err != nil {
				logrus.Fatal(err)
			}

			snapshots, err := uback.SortedListSnapshots(srcOpts.Source)
			if err != nil {
				logrus.Fatal(err)
			}

			snapshotsSet := make(map[uback.Snapshot]interface{})
			for _, s := range snapshots {
				snapshotsSet[s] = nil
			}

			forceFull := cmdBackupForceFull
			var fullInterval int
			var lastCommon *uback.Snapshot
			if !forceFull {
				if srcOpts.Options.String["StateFile"] == "" {
					logrus.Warn("StateFile option missing, full backup forced")
					forceFull = true
				} else if srcOpts.Options.String["FullInterval"] == "" {
					logrus.Warn("no interval between full backups given, full backup forced")
					forceFull = true
				} else {
					fullInterval, err = uback.ParseInterval(srcOpts.Options.String["FullInterval"])
					if err != nil {
						logrus.Fatal(err)
					}
				}
			}
			if !forceFull {
				var lastFull *uback.Backup
				for i, b := range backups {
					_, ok := snapshotsSet[b.Snapshot]
					if ok && lastCommon == nil {
						lastCommon = &backups[i].Snapshot
					}
					if b.BaseSnapshot == nil && lastFull == nil {
						lastFull = &backups[i]
					}
					if lastFull != nil && lastCommon != nil {
						break
					}
				}
				if lastFull == nil {
					logrus.Warn("no full backup found, full backup forced")
					forceFull = true
				} else if lastCommon == nil {
					logrus.Warn("no common snapshots found, full backup forced")
					forceFull = true
				} else {
					t, err := lastFull.Time()
					if err != nil {
						logrus.Fatal(err)
					}

					if time.Now().UTC().Sub(t).Seconds() >= float64(fullInterval)*0.9 {
						logrus.Printf("interval between full backups reached, full backup forced")
						forceFull = true
					}
				}
			}
			if forceFull {
				lastCommon = nil
			}

			backup, data, err := srcOpts.Source.CreateBackup(lastCommon)
			if err != nil {
				logrus.Fatal(err)
			}

			pr, pw := io.Pipe()
			cw, err := container.NewWriter(pw, &srcOpts.PublicKey, srcOpts.SourceType, compressionLevel)
			if err != nil {
				logrus.Fatal(err)
			}

			go func() {
				_, err := io.Copy(cw, data)
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				err = data.Close()
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				pw.CloseWithError(cw.Close())
			}()

			err = dstOpts.Destination.SendBackup(backup, pr)
			if err != nil {
				logrus.Fatal(err)
			}

			state := make(map[string]string)
			if srcOpts.Options.String["StateFile"] != "" {
				rawState, err := os.ReadFile(srcOpts.Options.String["StateFile"])
				if err != nil && !os.IsNotExist(err) {
					logrus.Fatal(err)
				}

				if rawState != nil {
					err = json.Unmarshal(rawState, &state)
					if err != nil {
						logrus.Fatal(err)
					}
				}

				state[dstOpts.Options.String["ID"]] = string(backup.Snapshot)
				rawState, err = json.Marshal(state)
				if err != nil {
					logrus.Fatal(err)
				}

				err = os.WriteFile(srcOpts.Options.String["StateFile"], rawState, 0o666)
				if err != nil {
					logrus.Fatal(err)
				}
			}

			if !cmdBackupNoPrune {
				err = uback.PruneSnapshots(srcOpts.Source, append([]uback.Snapshot{backup.Snapshot}, snapshots...), srcOpts.RetentionPolicies, state)
				if err != nil {
					logrus.Warnf("cannot prune snapshots: %v", err)
				}

				err = uback.PruneBackups(dstOpts.Destination, append([]uback.Backup{backup}, backups...), dstOpts.RetentionPolicies)
				if err != nil {
					logrus.Warnf("cannot prune backups: %v", err)
				}
			}
		},
	}
)

func init() {
	cmdBackup.Flags().BoolVarP(&cmdBackupForceFull, "force-full", "f", false, "force full backup")
	cmdBackup.Flags().BoolVarP(&cmdBackupNoPrune, "no-prune", "n", false, "do not prune snapshots and backups")
}
