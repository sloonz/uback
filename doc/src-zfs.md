# ZFS Source

Backup a zfs dataset using `zfs send`. Supports incremental backups.

Note : for restoration, target directory is only used to download the
backups. Where for another source you would use :

```shell
uback restore -d restore/ ...
```

Here you must use :

```shell
uback restore -o dataset=zpool/restore -d restore/ ...
```

The dataset must not exist at the start of the restoration process. If
you want to specify where it will be actually restored in the filesystem,
you can use this pattern :

```shell
zfs create -o mountpoint=/path/to/restore zpool/restore
uback restore -o dataset=zpool/restore/mydataset -d restore/ ...
```

Or, alternatively :

```shell
uback restore -o dataset=zpool/restore,receive-command="zfs receive -o mountpoint=/path/to/restore" -d restore/ ...
```

## Options

### Dataset

Required.

### Raw

Send a raw stream. See `zfs-send(8)`.

Optional, default: `true`

### Replicate

Make recursive snapshots, and send a replicate stream. See `zfs-send(8)`.

Optional, default: `false`

### UseBookmarks

Use ZFS bookmarks rather than snapshots as incremental bases. This uses
less disk space, but is incompatible with `Replicate: true`.

Optional, default: `true`

### Prefix

Prefix for snapshots/bookmarks names.

Optional, default: `"uback-"`

### Exclude

Exclude those datasets (comma-separated).

Optional, default: `""`

### @ListCommand

Optional, default: `[zfs list]`

### @SnapshotCommand

Optional, default: `[zfs snapshot]`

### @BookmarkCommand

Optional, default: `[zfs bookmark]`

### @SendCommand

Optional, default: `[zfs send]`

### @ReceiveCommand

Optional, default: `[zfs receive]`

### @DestroyCommand

Optional, default: `[zfs destroy]`

### @ReuseSnapshots

Optional.

Take a time interval (for example, `1h` for 1 hour). If set, if there
exists a snapshot that is more recent than that interval, then reuse
that snapshot for creating a backup rather than creating a new one. This
can be useful if you want to backup a single zfs filesystem to two
(or more) destinations.

Note that for this to work properly, the reuse interval must be smaller
that your interval between two backups to the same destination ;
otherwise, uback may use the same snapshot for your incremental base
and the reuse point, leading to an empty backup.