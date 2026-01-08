# ZFS Destination

A ZFS destination can only receive unencrypted backups from a ZFS
source. It will store the backup as a dataset rather than a backup file.

Note that since backups are stored as datasets, all backups on a ZFS
destination are considered as full backups even if they were sent as
incremental backups: the merge operation between the base backup and
the incremental one is done automatically when the destination receive
the backup.

## Options

### Dataset

Required.

### Raw

On restoration, send a raw stream. See `zfs-send(8)`.

Optional, default: `true`

### Replicate

On restoration, send a replicate stream. See `zfs-send(8)`.

Optional, default: `true`

### Prefix

Prefix for snapshots names.

Must match the Prefix option on your ZFS source.

Optional, default: `"uback-"`

### Exclude

Exclude those datasets (comma-separated).

Optional, default: `""`

### @SendCommand

Optional, defaults: `[zfs send]`

### @ReceiveCommand

Optional, defaults: `[zfs receive]`

### @DestroyCommand

Optional, defaults: `[zfs destroy]`
