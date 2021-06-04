package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"filippo.io/age"
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
		id, err := age.GenerateX25519Identity()
		if err != nil {
			logrus.Fatal(err)
		}

		if len(args) > 0 {
			err = os.WriteFile(args[0], []byte(fmt.Sprintf("%s\n", id.String())), 0600)
			if err != nil {
				logrus.Fatal(err)
			}
		} else {
			fmt.Println(id.String())
		}

		if len(args) > 1 {
			err = os.WriteFile(args[1], []byte(fmt.Sprintf("%s\n", id.Recipient().String())), 0666)
			if err != nil {
				logrus.Fatal(err)
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
		var identitiesData []byte
		if len(args) > 0 {
			identitiesData, err = os.ReadFile(args[0])
		} else {
			identitiesData, err = io.ReadAll(os.Stdin)
		}
		if err != nil {
			logrus.Fatal(err)
		}

		identities, err := age.ParseIdentities(bytes.NewBuffer(identitiesData))
		if err != nil {
			logrus.Fatal(err)
		}
		if len(identities) != 1 {
			logrus.Fatalf("unexpected number of identities: %d", len(identities))
		}

		identity, ok := identities[0].(*age.X25519Identity)
		if !ok {
			logrus.Fatalf("unexpected identity type: %v", reflect.TypeOf(identities[0]))
		}

		if len(args) > 1 {
			err = os.WriteFile(args[1], []byte(fmt.Sprintf("%s\n", identity.Recipient().String())), 0666)
			if err != nil {
				logrus.Fatal(err)
			}
		} else {
			fmt.Println(identity.Recipient().String())
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
