package uback

import (
	"io"
)

// Represents a snapshot. Should be in the YYYYMMDDTHHMMSS.MMM format.
type Snapshot string

// Represents a backup
type Backup struct {
	// Snapshot from which this backup was created
	Snapshot

	// If nil, this backup is a full backup.
	// If not nil, this is an incremental backup whose base is BsaeSnapshot
	BaseSnapshot *Snapshot
}

// A source is what we want to backup
type Source interface {
	// Archives can be used for local backup restoration in addition to incremental bases.
	// They are subject to retention rules.
	ListArchives() ([]Snapshot, error)
	RemoveArchive(snapshot Snapshot) error

	// Bookmarks can be used as incremental bases, and are only kept around for them.
	// They are removed as soon as no destination needs them for creating an incremental.
	ListBookmarks() ([]Snapshot, error)
	RemoveBookmark(snapshot Snapshot) error

	// Create a backup from a base snapshot (if supported by the source), or nil for a full snapshot.
	// Returns the name of the created backup, and a reader to the backup data.
	// The returned backup can be full even if baseSnapshot is not nil, but cannot
	// be an incremental backup if baseSnapshot is nil.
	CreateBackup(baseSnapshot *Snapshot) (Backup, io.ReadCloser, error)

	// Restore a backup whose name is `backup` and data is `data` onto the `target` directory.
	// If `backup` is an incremental backup, his base is guaranteed to already have been restored.
	RestoreBackup(target string, backup Backup, data io.Reader) error
}

// A destination is a storage for backups
type Destination interface {
	// List backups present on the source
	ListBackups() ([]Backup, error)

	// Remove a backup
	RemoveBackup(backup Backup) error

	// Store a backup whose data is `data`
	SendBackup(backup Backup, data io.Reader) error

	// Retrieve the content of a previously stored backup
	ReceiveBackup(backup Backup) (io.ReadCloser, error)
}
