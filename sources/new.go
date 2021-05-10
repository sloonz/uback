package sources

import (
	"github.com/sloonz/uback/lib"

	"fmt"
)

// Create a new source from options ; you should be able to call everything on the returned interface
func New(options *uback.Options) (uback.Source, error) {
	switch options.String["Type"] {
	case "tar":
		return newTarSource(options)
	case "mariabackup":
		return newMariaBackupSource(options)
	default:
		return nil, fmt.Errorf("invalid source type %v", options.String["Type"])
	}
}

// Create a new source only from its type ; you should be able to call only RestoreBackup on the returned interface
func NewForRestoration(typ string) (uback.Source, error) {
	switch typ {
	case "tar":
		return newTarSourceForRestoration()
	case "mariabackup":
		return newMariaBackupSourceForRestoration()
	default:
		return nil, fmt.Errorf("invalid source type %v", typ)
	}
}
