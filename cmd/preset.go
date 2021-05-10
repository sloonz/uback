package cmd

import (
	"github.com/sloonz/uback/lib"

	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cmdPreset = &cobra.Command{
	Use:   "preset",
	Short: "Manage presets",
}

var presetSetClear bool
var cmdPresetSet = &cobra.Command{
	Use:   "set <preset-name> [option=value...]",
	Short: "Create or modify preset",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := os.MkdirAll(presetsDir, 0777)
		if err != nil {
			logrus.Fatal(err)
		}

		presetPath := path.Join(presetsDir, fmt.Sprintf("%v.json", args[0]))

		var kvs []uback.KeyValuePair
		if !presetSetClear {
			data, err := os.ReadFile(presetPath)
			if err != nil && !os.IsNotExist(err) {
				logrus.Fatal(err)
			} else if err == nil {
				err = json.Unmarshal(data, &kvs)
				if err != nil {
					logrus.Fatal(err)
				}
			}
		}

		for _, opts := range args[1:] {
			kvs = append(kvs, uback.SplitOptions(opts)...)
		}

		data, err := json.Marshal(kvs)
		if err != nil {
			logrus.Fatal(err)
		}

		err = os.WriteFile(presetPath, data, 0666)
		if err != nil {
			logrus.Fatal(err)
		}
	},
}

var cmdPresetRemove = &cobra.Command{
	Use:   "remove <preset-name...>",
	Short: "Remove presets",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, name := range args {
			err := os.Remove(path.Join(presetsDir, fmt.Sprintf("%s.json", name)))
			if err != nil && !os.IsNotExist(err) {
				logrus.Warn(err)
			}
		}
	},
}

var presetListVerbose bool
var cmdPresetList = &cobra.Command{
	Use:   "list",
	Short: "List presets",
	Run: func(cmd *cobra.Command, args []string) {
		for name, options := range presets {
			if presetListVerbose {
				fmt.Printf("%v %v\n", name, options)
			} else {
				fmt.Printf("%v\n", name)
			}
		}
	},
}

var cmdPresetEval = &cobra.Command{
	Use:   "eval <option-line>",
	Short: "Show the evaluated (after presets substitutions and template evaluation) option line",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kvs := uback.SplitOptions(args[0])
		options, err := uback.EvalOptions(kvs, presets)
		if err != nil {
			logrus.Fatal(err)
		}

		for k, v := range options.String {
			fmt.Printf("%s: %s\n", k, v)
		}
		for k, v := range options.StrSlice {
			fmt.Printf("@%s: %v\n", k, v)
		}
	},
}

func init() {
	cmdPresetList.Flags().BoolVarP(&presetListVerbose, "verbose", "v", false, "also print preset content")
	cmdPresetSet.Flags().BoolVarP(&presetSetClear, "clear", "c", false, "remove existing entries")
	cmdPreset.AddCommand(cmdPresetSet, cmdPresetRemove, cmdPresetList, cmdPresetEval)
}
