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

### @ReuseSnapshots

Optional.

Take a time interval (for example, `1h` for 1 hour). If set, if there
exists a snapshot that is more recent than that interval, then reuse
that snapshot for creating a backup rather than creating a new one. This
can be useful if you want to backup a single btrfs filesystem to two
(or more) destinations.

Note that for this to work properly, the reuse interval must be smaller
that your interval between two backups to the same destination ;
otherwise, uback may use the same snapshot for your incremental base
and the reuse point, leading to an empty backup.
