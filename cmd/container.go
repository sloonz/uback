package cmd

import (
	"github.com/sloonz/uback/container"
	"github.com/sloonz/uback/lib"

	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cmdContainerType = &cobra.Command{
	Use:   "type [file]",
	Short: "Prints the source type of a backup file (if omitted: stdin)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var f io.ReadCloser
		if len(args) == 0 {
			f = io.NopCloser(os.Stdin)
		} else {
			f, err = os.Open(args[0])
			if err != nil {
				logrus.Fatal(err)
			}
			defer f.Close()
		}

		r, err := container.NewReader(f)
		if err != nil {
			logrus.Fatal(err)
		}

		fmt.Printf("%s\n", r.Options.String["Type"])
	},
}

var cmdContainerExtractKeyFile string
var cmdContainerExtractKey string
var cmdContainerExtract = &cobra.Command{
	Use:   "extract [input-file] [output-file]",
	Args:  cobra.MaximumNArgs(2),
	Short: "Print the decrypted and decompressed content of the backup file",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var in io.ReadCloser
		if len(args) == 0 {
			in = io.NopCloser(os.Stdin)
		} else {
			in, err = os.Open(args[0])
			if err != nil {
				logrus.Fatal(err)
			}
			defer in.Close()
		}

		var out io.Writer
		if len(args) <= 1 {
			out = os.Stdout
		} else {
			out, err := os.OpenFile(args[1], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
			if err != nil {
				logrus.Fatal(err)
			}
			defer out.Close()
		}

		r, err := container.NewReader(in)
		if err != nil {
			logrus.Fatal(err)
		}

		sk, err := uback.LoadPrivateKey(cmdContainerExtractKeyFile, cmdContainerExtractKey)
		if err != nil {
			logrus.Fatal(err)
		}

		err = r.Unseal(sk)
		if err != nil {
			logrus.Fatal(err)
		}

		_, err = io.Copy(out, r)
		if err != nil {
			logrus.Fatal(err)
		}
	},
}

var cmdContainerCreateCompressionLevel int
var cmdContainerCreateKeyFile string
var cmdContainerCreateKey string
var cmdContainerCreate = &cobra.Command{
	Use:   "create <type> [input file] [output-file]",
	Args:  cobra.RangeArgs(1, 3),
	Short: "Create a backup file",
	Run: func(cmd *cobra.Command, args []string) {
		typ := args[0]

		var err error
		var in io.ReadCloser
		if len(args) <= 1 {
			in = io.NopCloser(os.Stdin)
		} else {
			in, err = os.Open(args[1])
			if err != nil {
				logrus.Fatal(err)
			}
			defer in.Close()
		}

		var out io.Writer
		if len(args) <= 2 {
			out = os.Stdout
		} else {
			out, err := os.OpenFile(args[2], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
			if err != nil {
				logrus.Fatal(err)
			}
			defer out.Close()
		}

		pk, err := uback.LoadPublicKey(cmdContainerCreateKeyFile, cmdContainerCreateKey)
		if err != nil {
			logrus.Fatal(err)
		}

		w, err := container.NewWriter(out, pk, typ, cmdContainerCreateCompressionLevel)
		if err != nil {
			logrus.Fatal(err)
		}

		_, err = io.Copy(w, in)
		if err != nil {
			logrus.Fatal(err)
		}

		err = w.Close()
		if err != nil {
			logrus.Fatal(err)
		}
	},
}

var cmdContainer = &cobra.Command{
	Use:   "container",
	Short: "Directly manipulate uback files",
}

func init() {
	cmdContainer.AddCommand(cmdContainerType, cmdContainerExtract, cmdContainerCreate)
	cmdContainerExtract.Flags().StringVarP(&cmdContainerExtractKeyFile, "key-file", "k", "", "private key file for decryption")
	cmdContainerExtract.Flags().StringVarP(&cmdContainerExtractKey, "key", "K", "", "private key for decryption")
	cmdContainerCreate.Flags().StringVarP(&cmdContainerCreateKeyFile, "key-file", "k", "", "public key file for encryption")
	cmdContainerCreate.Flags().StringVarP(&cmdContainerCreateKey, "key", "K", "", "public key for encryption")
	cmdContainerCreate.Flags().IntVarP(&cmdContainerCreateCompressionLevel, "compression-level", "z", 3, "compression level")
}
