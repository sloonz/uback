package sources

import (
	"github.com/sloonz/uback/lib"

	"fmt"
	"strings"
)

// Create a new source from options ; you should be able to call everything on the returned interface
func New(options *uback.Options) (src uback.Source, typ string, err error) {
	typ = options.String["Type"]
	switch typ {
	case "tar":
		src, err = newTarSource(options)
	case "mariabackup":
		src, err = newMariaBackupSource(options)
	case "command":
		src, typ, err = newCommandSource(options)
	default:
		return nil, "", fmt.Errorf("invalid source type %v", options.String["Type"])
	}
	return
}

// Create a new source only from its type ; you should be able to call only RestoreBackup on the returned interface
func NewForRestoration(typ string) (uback.Source, error) {
	switch typ {
	case "tar":
		return newTarSourceForRestoration()
	case "mariabackup":
		return newMariaBackupSourceForRestoration()
	default:
		if strings.HasPrefix(typ, "command:") {
			return newCommandSourceForRestoration(typ[len("command:"):])
		}
		return nil, fmt.Errorf("invalid source type %v", typ)
	}
}
