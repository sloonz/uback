# btrfs Destination

A btrfs destination can only receive unencrypted backups from a btrfs
source. It will store the backup as a btrfs subvolume rather than a
backup file.

Note that since backups are stored as btrfs subvolumes, all backups on a
btrfs destination are considered as full backups even if they were sent
as incremental backups: the merge operation between the base backup and
the incremental one is done automatically when the destination receive
the backup.

## Options

### Path

Required.

Note: must be in a btrfs filesystem.

### @SnapshotCommand

Optional, defaults: `[btrfs subvolume snapshot]`

### @SendCommand

Optional, defaults: `[btrfs send]`

### @ReceiveCommand

Optional, defaults: `[btrfs receive]`

### @DeleteCommand

Optional, defaults: `[btrfs subvolume delete]`
