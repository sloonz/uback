package cmd

import (
	"github.com/sloonz/uback/x25519"

	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cmdKeyGen = &cobra.Command{
	Use:   "gen [private-key-file] [public-key-file]",
	Short: "Create a keypair used for encrypting backups",
	Args:  cobra.MaximumNArgs(2),
	Long: strings.TrimSpace(`
Create a new keypair. If no argument is given, output the private key
on standard output. If only one argument is given, write the private
key in a file given by the first argument. If both arguments are given,
write the private key in a file given by the first argument and the
public key in a file given by the second argument.
	`),
	Run: func(cmd *cobra.Command, args []string) {
		pubKey, privKey, err := x25519.GenerateKey()
		if err != nil {
			logrus.Fatal(err)
		}

		pubKeyDer, err := pubKey.Marshal()
		if err != nil {
			logrus.Fatal(err)
		}

		privKeyDer, err := privKey.Marshal()
		if err != nil {
			logrus.Fatal(err)
		}

		pubKeyPem := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyDer,
		})

		privKeyPem := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privKeyDer,
		})

		if len(args) == 0 {
			fmt.Print(string(privKeyPem))
		} else {
			err := os.WriteFile(args[0], privKeyPem, 0600)
			if err != nil {
				logrus.Fatal(err)
			}

			if len(args) == 2 {
				err := os.WriteFile(args[1], pubKeyPem, 0666)
				if err != nil {
					logrus.Fatal(err)
				}
			}
		}
	},
}

var cmdKeyPub = &cobra.Command{
	Use:   "pub [private-key-file] [public-key-file]",
	Short: "Extract public key from the private key",
	Args:  cobra.MaximumNArgs(2),
	Long: strings.TrimSpace(`
Extract the public key. If no argument is given, take private key from
stdin and print public key on stdout. If only one argument is given,
take private key from the file given by the first argument, and print
public key on stdout.
	`),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var privKeyPem []byte

		if len(args) == 0 {
			privKeyPem, err = io.ReadAll(os.Stdin)
			if err != nil {
				logrus.Fatal(err)
			}
		} else {
			privKeyPem, err = os.ReadFile(args[0])
			if err != nil {
				logrus.Fatal(err)
			}
		}

		privKeyDer, _ := pem.Decode(privKeyPem)
		privKey, err := x25519.ParsePrivateKey(privKeyDer.Bytes)
		if err != nil {
			logrus.Fatal(err)
		}

		pubKey, err := privKey.Public()
		if err != nil {
			logrus.Fatal(err)
		}

		pubKeyDer, err := pubKey.Marshal()
		if err != nil {
			logrus.Fatal(err)
		}

		pubKeyPem := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyDer,
		})

		if len(args) == 2 {
			err = os.WriteFile(args[1], pubKeyPem, 0666)
			if err != nil {
				logrus.Fatal(err)
			}
		} else {
			fmt.Print(string(pubKeyPem))
		}
	},
}

var cmdKey = &cobra.Command{
	Use:   "key",
	Short: "Encryption keys management",
}

func init() {
	cmdKey.AddCommand(cmdKeyGen, cmdKeyPub)
}
