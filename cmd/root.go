package cmd

import (
	"github.com/sloonz/uback/lib"

	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	presetsDir string
	logLevel   string
	presets    map[string][]uback.KeyValuePair

	tag       = "git"
	commit    = "unknown"
	buildDate = "unknown"

	rootCmd    = &cobra.Command{Use: "uback"}
	cmdVersion = &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\n", tag)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Build Date: %s\n", buildDate)
		},
	}
)

func init() {
	cobra.OnInitialize(func() {
		var err error

		if presetsDir == "" {
			usr, err := user.Current()
			if err != nil {
				logrus.Fatal(err)
			}

			if usr.Uid == "0" {
				presetsDir = path.Join("/etc", "uback", "presets")
			} else {
				presetsDir = path.Join(usr.HomeDir, ".config", "uback", "presets")
			}

			if logLevel != "" {
				level, err := logrus.ParseLevel(logLevel)
				if err == nil {
					logrus.SetLevel(level)
				} else {
					logrus.Warnf("Cannot set log level: %v", err)
				}
			}
		}

		presets, err = uback.ReadPresets(presetsDir)
		if err != nil {
			logrus.Fatal(err)
		}
	})

	rootCmd.PersistentFlags().StringVarP(&presetsDir, "presets-dir", "p", "", "path to presets directory")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "", os.Getenv("LOG_LEVEL"), "log level (trace, debug, info, warn, error)")
	rootCmd.AddCommand(cmdPreset, cmdBackup, cmdKey, cmdContainer, cmdList, cmdPrune, cmdFetch, cmdRestore, cmdVersion, cmdProxy)
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logrus.Fatal(err)
	}
}
