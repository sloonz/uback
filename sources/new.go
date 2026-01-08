package sources

import (
	"fmt"
	"strings"

	"github.com/google/shlex"
	uback "github.com/sloonz/uback/lib"
)

// Create a new source from options ; you should be able to call everything on the returned interface
func New(options *uback.Options) (src uback.Source, typ string, err error) {
	typ = options.String["Type"]
	switch typ {
	case "btrfs":
		src, err = newBtrfsSource(options)
	case "zfs":
		src, err = newZfsSource(options)
	case "tar":
		src, err = newTarSource(options)
	case "mariabackup":
		src, err = newMariaBackupSource(options)
	case "command":
		src, typ, err = newCommandSource(options)
	case "proxy":
		src, typ, err = newProxySource(options)
	default:
		return nil, "", fmt.Errorf("invalid source type %v", options.String["Type"])
	}
	return
}

// Create a new source only from its type ; you should be able to call only RestoreBackup on the returned interface
func NewForRestoration(options *uback.Options, typ string) (uback.Source, error) {
	switch typ {
	case "btrfs":
		return newBtrfsSourceForRestoration(options)
	case "zfs":
		return newZfsSourceForRestoration(options)
	case "tar":
		return newTarSourceForRestoration()
	case "mariabackup":
		return newMariaBackupSourceForRestoration(options)
	case "proxy":
		return nil, ErrProxyNoRestoration
	default:
		if strings.HasPrefix(typ, "command:") {
			command, err := shlex.Split(typ[len("command:"):])
			if err != nil {
				return nil, err
			}
			return newCommandSourceForRestoration(command, options)
		}
		return nil, fmt.Errorf("invalid source type %v", typ)
	}
}
