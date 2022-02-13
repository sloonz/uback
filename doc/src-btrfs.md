# btrfs Source

Backup a btrfs volume or subvolume using `btrfs send`. Supports
incremental backups.

## Options

### SnapshotsPath

Required.

Note: `SnapshotsPath` and `Path` both must be on the same btrfs
filesystem.

### Path

Required.

Note: This must be a btrfs subvolume.

### @SnapshotCommand

Optional, defaults: `[btrfs subvolume snapshot]`

### @SendCommand

Optional, defaults: `[btrfs send]`

### @ReceiveCommand

Optional, defaults: `[btrfs receive]`

### @DeleteCommand

Optional, defaults: `[btrfs subvolume delete]`
