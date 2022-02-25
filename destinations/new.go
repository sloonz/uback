package destinations

import (
	"github.com/sloonz/uback/lib"

	"fmt"
)

func New(options *uback.Options) (uback.Destination, error) {
	switch options.String["Type"] {
	case "btrfs":
		return newBtrfsDestination(options)
	case "fs":
		return newFSDestination(options)
	case "object-storage":
		return newObjectStorageDestination(options)
	case "command":
		return newCommandDestination(options)
	default:
		return nil, fmt.Errorf("invalid destination type %v", options.String["Type"])
	}
}
