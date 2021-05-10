# tar Source

Backup a part of the filesystem using tar. Supports incremental backups.

## Limitations

Incremental backups do not support file deletion : if you delete a file,
then make an incremental backup, and then restore the backup, the deleted
file will still appear in the restored filesystem if it was present in
the full backup/in a previous incremental backup.

## Options

### SnapshotsPath

Optional, but required for incremental backups.

### Path

Required.

### @Command

Optional, defaults: `[tar]`
