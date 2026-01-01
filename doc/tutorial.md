# Tutorial

## First Backup

Let’s make our first backup. We have to create our keypair for
encryption/decrytion first:

```
$ uback key gen backup.key backup.pub
```

`backup.key` is the private key, used only for restoring your backups. You
should keep it in a safe (losing it means losing all your backups), secure
(disclosing it means anyone that know it can decrypt your backups) place,
preferably offline. `backup.pub` is the public key, used for creating
the backups. You can recreate it from the private key if you lose it ;
it cannot be used for decrypting your backups, so it is not important
to keep it secret.

They are standard [age](https://age-encryption.org/) keys ; you can use
`age-keygen` to generate/manage them.

Then, let’s create stuff we want to backup :

```
$ mkdir /tmp/etc
$ echo hello > /tmp/etc/a
```

We can now create our encrypted backup :

```
$ uback backup type=tar,path=/tmp/etc,key-file=backup.pub type=fs,path=/tmp/my-backups/etc/
WARN[0000] SnapshotsPath option missing, incremental backups will be impossible  source=tar
WARN[0000] StateFile option missing, full backup forced 
INFO[0000] creating backup: 20210515T124130.706-full.ubkp  source=tar
INFO[0000] running: /usr/bin/tar --create -C /tmp/etc . 
INFO[0000] writing backup to /tmp/my-backups/etc/_tmp-20210515T124130.706-full.ubkp  destination=fs
INFO[0000] moving final backup to /tmp/my-backups/etc/20210515T124130.706-full.ubkp  destination=fs
INFO[0000] deleting bookmark                             bookmark=20210515T124130.706
WARN[0000] no retention policies set for destination, keeping everything 
```

The three warnings will be addressed in the next sections.

Most options here are pretty much self-explanatory. But to state the
obvious, `path=/tmp/etc` specifies what we’re backing up, `type=tar` how we
bundle it, `key-file` is the public key use for encrypting the backup,
`path=/tmp/my-backups/etc` is the location of the backups, and `type=fs`
means that we’re just storing the backups as plain files on the local
filesystem. The `id` of the destination is used for bookkeeping purposes ;
it must be unique for each source-destination pair (in other words,
if you backup a source to multiple destinations, each destination must
have a different ID).

Note that backups are directly saved to the destination `path` (you
can check by running `ls /tmp/my-backups/etc`) ; there is nothing to
identify the source except the `etc` name we manually put in the path
configuration of the destination. If you want to save different sources
(for example `/etc` and `/var/lib`), you should put them in different
paths in the destination too, otherwise backups of different sources will
be intermingled. This a general feature of destinations ; for example,
if you want to backup different souces on a same S3-compatible object
storage, you should put them in different buckets or give them different
prefixes if they are in the same bucket.

We can see that the backup has been created :

```
$ uback list backups type=fs,path=/tmp/my-backups/etc/
20210515T124130.706 (full)
```

We can try to restore it :

```
$ uback restore -d /tmp/restored-etc/ type=fs,path=/tmp/my-backups/etc/,key-file=backup.key
INFO[0000] restoring 20210515T124130.706-full.ubkp onto /tmp/restored-etc/ 
INFO[0000] running /usr/bin/tar -x -C /tmp/restored-etc/20210515T124130.706  source=tar
```

And we can see that the backup has been restored :

```
$ cat /tmp/restored-etc/20210515T124130.706/a
hello
```

## Incremental Backups

Let’s go back to the two warnings we got in the previous section :

```
WARN[0000] SnapshotsPath option missing, incremental backups will be impossible  source=tar
WARN[0000] StateFile option missing, full backup forced 
```

`tar` supports incremental archives (if you’re curious, see the
`--listed-incremental` option in the `tar(1)` manual). `uback` can make
use of this feature to create incremental backups.

To support incremental backups in `uback`, we need to provide two options
to the source :

* The snapshots directory, where `uback` stores the information required
to create a new incremental backup from a previous made one. This option
is specific to the source type (here, `tar`) : different sources types
may have different way to store this information. For the  `tar` source
type, this is given by the `snapshots-path` option.

* The state file, where `uback` keeps tracks of the last snapshot on
each destination, allowing us to remove unused ones. This is not specific
to the source type and is needed as soon as you want incremental
backups. It is given by the `state-file` option.

* Note that for sources, there is two distinct kind of snapshots, archives
and bookmarks. Both can be used as incremental bases to create incremental
backups, but bookmarks only store minimal metadata necessary to know
what delta must be sent in the incremental backup and cannot be used as
a local backup, whereas archives can also be used as a local backup.

First, let’s create a (relatively) big file (so we can see the effect
of incremental backups), a full backup and its associated snapshot :

```
$ dd if=/dev/urandom bs=1M count=10 of=/tmp/etc/BIG
$ uback backup type=tar,path=/tmp/etc,key-file=backup.pub,state-file=/tmp/uback/state/etc.json,snapshots-path=/tmp/uback/tar-snapshots/etc/,full-interval=weekly id=fs,type=fs,path=/tmp/my-backups/etc/
WARN[0000] no common snapshots found, full backup forced 
INFO[0000] creating backup: 20210515T130148.562-full.ubkp  source=tar
INFO[0000] running: /usr/bin/tar --create --listed-incremental=/tmp/uback/tar-snapshots/etc/_tmp-20210515T130148.562 -C /tmp/etc . 
INFO[0000] writing backup to /tmp/my-backups/etc/_tmp-20210515T130148.562-full.ubkp  destination=fs
INFO[0000] moving final backup to /tmp/my-backups/etc/20210515T130148.562-full.ubkp  destination=fs
WARN[0000] no retention policies set for destination, keeping everything 
```

(note that because we didn’t specify a `snapshots-path` in the previous
section, we can’t reuse the full backup created then and have to create
a brand new one.)

Then let’s change the content of `a` and create an incremental backup :

```
$ echo hello world > /tmp/etc/a
$ uback backup type=tar,path=/tmp/etc,key-file=backup.pub,state-file=/tmp/uback/state/etc.json,snapshots-path=/tmp/uback/tar-snapshots/etc/,full-interval=weekly id=fs,type=fs,path=/tmp/my-backups/etc/
INFO[0000] creating backup: 20210515T130258.315-from-20210515T130148.562.ubkp  source=tar
INFO[0000] running: /usr/bin/tar --create --listed-incremental=/tmp/uback/tar-snapshots/etc/_tmp-20210515T130258.315 -C /tmp/etc . 
INFO[0000] writing backup to /tmp/my-backups/etc/_tmp-20210515T130258.315-from-20210515T130148.562.ubkp  destination=fs
INFO[0000] moving final backup to /tmp/my-backups/etc/20210515T130258.315-from-20210515T130148.562.ubkp  destination=fs
INFO[0000] deleting bookmark                             bookmark=20210515T130148.562
WARN[0000] no retention policies set for destination, keeping everything 
```

Notice the name of the backup,
`20210515T130258.315-from-20210515T130148.562.ubkp`, which indicates
that an incremental backup has indeed been created. We can look at the
size of the incremental backup, to check that it does not contains the
`BIG` file which has not been modified between the two backups :

```
$ ls -lh /tmp/my-backups/etc/
total 11M
-rw-r--r-- 1 user user 260 May 15 14:41 20210515T124130.706-full.ubkp
-rw-r--r-- 1 user user 11M May 15 15:01 20210515T130148.562-full.ubkp
-rw-r--r-- 1 user user 231 May 15 15:02 20210515T130258.315-from-20210515T130148.562.ubkp
```

Note the `full-interval` option : it controls the time between two full
backups. More precisely, a full backup will be created instead of an
incremental one if the age of the most recent full backup is bigger than
the time specified in `full-interval`. `full-interval` can be :

* `10` for 10 seconds,
* `10h` for 10 hours,
* `10d` for 10 days (10*24 hours),
* `10w` for 10 weeks (10*7 days),
* `10m` for 10*30 days,
* `10y` for 10*365 days,
* `hourly` is an alias for `1h`
* `daily` is an alias for `1d`
* `weekly` is an alias for `1w`
* `monthly` is an alias for `1m`
* `yearly` is an alias for `1y`

Finally, let’s restore the last backup to check that the `BIG` file will
be properly restored even if not present in the last incremental backup :

```
$ sha256sum /tmp/etc/BIG
903803469a5c720b22c388853c92abba1b69c5f351a625ffdab41e06b493cd7a  /tmp/etc/BIG
$ uback restore -d /tmp/restored-etc/ type=fs,path=/tmp/my-backups/etc/,key-file=backup.key
INFO[0000] restoring 20210515T130148.562-full.ubkp onto /tmp/restored-etc/ 
INFO[0000] running /usr/bin/tar -x -C /tmp/restored-etc/20210515T130148.562  source=tar
INFO[0000] restoring 20210515T130258.315-from-20210515T130148.562.ubkp onto /tmp/restored-etc/ 
INFO[0000] running /usr/bin/tar -x -C /tmp/restored-etc/20210515T130258.315  source=tar
$ sha256sum /tmp/restored-etc/20210515T130258.315/BIG
903803469a5c720b22c388853c92abba1b69c5f351a625ffdab41e06b493cd7a  /tmp/restored-etc/20210515T130258.315/BIG
```

## Presets and Templates

The command lines in the previous were quite mouthful. And we haven’t
introduced retention policies yet, which will make even more bigger ! To
keep them under control, `uback` has a system of presets, that allow
you to define a set of options and re-use them.

First, let’s make a big simple preset that we can use for our source :

```
$ uback preset set etc-src type=tar,path=/tmp/etc,key-file=backup.pub,state-file=/tmp/uback/state/etc.json,snapshots-path=/tmp/uback/tar-snapshots/etc/,full-interval=weekly
$ uback preset list -v
etc-src [[Type tar] [Path /tmp/etc] [KeyFile backup.pub] [StateFile /tmp/uback/state/etc.json] [SnapshotsPath /tmp/uback/tar-snapshots/etc/] [FullInterval weekly]]
$ uback backup preset=etc-src id=fs,type=fs,path=/tmp/my-backups/etc/
INFO[0000] creating backup: 20210515T131759.943-from-20210515T130258.315.ubkp  source=tar
INFO[0000] running: /usr/bin/tar --create --listed-incremental=/tmp/uback/tar-snapshots/etc/_tmp-20210515T131759.943 -C /tmp/etc . 
INFO[0000] writing backup to /tmp/my-backups/etc/_tmp-20210515T131759.943-from-20210515T130258.315.ubkp  destination=fs
INFO[0000] moving final backup to /tmp/my-backups/etc/20210515T131759.943-from-20210515T130258.315.ubkp  destination=fs
INFO[0000] deleting bookmark                             bookmark=20210515T130258.315
WARN[0000] no retention policies set for destination, keeping everything 
```

A preset is a list of key-value pairs that store options. We can see that
internally, options in `uback` are formatted in `PascalCase`. The case
conversion is made automatically when you specify options on the command
line ; however, if you want to write your own sources/destinations or
want to manipulate the preset files directly, it is something you want
to keep in mind.

Presets are stored as simple JSON files that you can manipulate manually
or from your own scripts :

```
$ cat $HOME/.config/uback/presets/etc-src.json
[["Type","tar"],["Path","/tmp/etc"],["KeyFile","backup.pub"],["StateFile","/tmp/uback/state/etc.json"],["SnapshotsPath","/tmp/uback/tar-snapshots/etc/"],["FullInterval","weekly"]]
```

Presets can reference other presets, so we can mare a generic `src`
preset that will define the `KeyFile` and `FullInterval` options, a
`tar-src` preset that will reference it, and build our `etc-src` from it :

```
$ uback preset remove etc-src
$ uback preset set src key-file=backup.pub,full-interval=weekly
$ uback preset set tar-src type=tar,preset=src
$ uback preset set etc-src preset=tar-src,path=/tmp/etc,state-file=/tmp/uback/state/etc.json,snapshots-path=/tmp/uback/tar-snapshots/etc/
$ uback preset list -v
etc-src [[Preset tar-src] [Path /tmp/etc] [StateFile /tmp/uback/state/etc.json] [SnapshotsPath /tmp/uback/tar-snapshots/etc/]]
src [[KeyFile backup.pub] [FullInterval weekly]]
tar-src [[Type tar] [Preset src]]
$ uback preset eval preset=etc-src
KeyFile: backup.pub
FullInterval: weekly
Path: /tmp/etc
StateFile: /tmp/uback/state/etc.json
SnapshotsPath: /tmp/uback/tar-snapshots/etc/
Type: tar
```

(Note that we have specified `backup.pub` as a relative file. In a real
setup, you should really use the full absolute path for it.)

It’s nice, but it would be even nicer to be somewhat able to put
the `StateFile` option in the `src` preset and the `SnapshotsPath`
in the `tar-src` preset, but in a modular way so other sources can
make use of those presets. The good news is: since values are [Go
templates](https://golang.org/pkg/text/template/), it is indeed possible :

```
$ uback preset remove etc-src src tar-src
$ uback preset set src key-file=backup.pub,full-interval=weekly,state-file=/tmp/uback/state/{{.Path}}.json
$ uback preset set tar-src type=tar,preset=src,snapshots-path=/tmp/uback/tar-snapshots/{{.Path}}
$ uback preset set etc-src preset=tar-src,path=/tmp/etc
$ uback preset list -v
src [[KeyFile backup.pub] [FullInterval weekly] [StateFile /tmp/uback/state/{{.Path}}.json]]
tar-src [[Type tar] [Preset src] [SnapshotsPath /tmp/uback/tar-snapshots/{{.Path}}]]
etc-src [[Preset tar-src] [Path /tmp/etc]]
$ uback preset eval preset=etc-src
Path: /tmp/etc
Type: tar
KeyFile: backup.pub
FullInterval: weekly
StateFile: /tmp/uback/state/<no value>.json
SnapshotsPath: /tmp/uback/tar-snapshots/<no value>
```

Oops ! something went wrong here. What happened ? `tar-src` and `src`
presets made use of the `Path` option, but were included before the
option was set. Let’s fix that :

```
$ uback preset remove etc-src
$ uback preset set etc-src path=/tmp/etc,preset=tar-src
$ uback preset eval preset=etc-src
Path: /tmp/etc
Type: tar
KeyFile: backup.pub
FullInterval: weekly
StateFile: /tmp/uback/state//tmp/etc.json
SnapshotsPath: /tmp/uback/tar-snapshots//tmp/etc
```

Far better ! But there is still one issue here, which is that the path is
inserted verbatim in the `StateFile` and `SnapshotsPath` options. What if
we don't want a deep hierarchy in `/tmp/uback/tar-snapshots/` ? What if
we set `path` to `/tmp/etc/` ? Then the `StateFile` option will be set to
`/tmp/uback/state//tmp/etc/.json`, which can be surprising.

Thankfully, there is a solution. Values are Go templates that includes
[Sprig Functions](https://masterminds.github.io/sprig/).

First, let’s change `src` : it no longer uses the `Path` option,
but must be provided an `SourceID` option :

```
$ uback preset remove src
$ uback preset set src key-file=backup.pub,full-interval=weekly,state-file=/tmp/uback/state/{{.SourceID}}.json
```

Then, let’s make `tar-src` make that `SourceID` option from the `Path`
option, and use it itself for its `SnapshotsPath` option :

```
$ uback preset remove tar-src
$ uback preset set tar-src 'type=tar,source-id={{.Path|clean|replace "/" "-"|trimSuffix "-"}},preset=src,snapshots-path=/tmp/uback/tar-snapshots/{{.SourceID}}'
$ uback preset eval preset=etc-src
SourceID: -tmp-etc
KeyFile: backup.pub
FullInterval: weekly
StateFile: /tmp/uback/state/-tmp-etc.json
SnapshotsPath: /tmp/uback/tar-snapshots/-tmp-etc
Path: /tmp/etc
Type: tar
```

Finally, let’s just double-check that we can reuse those preset for other sources :

```
$ uback preset eval path=/tmp/var/lib,preset=tar-src
SourceID: -tmp-var-lib
KeyFile: backup.pub
FullInterval: weekly
StateFile: /tmp/uback/state/-tmp-var-lib.json
SnapshotsPath: /tmp/uback/tar-snapshots/-tmp-var-lib
Path: /tmp/var/lib
Type: tar
```

## Retention Policy

Let’s now tackle the last remaining warning :

```
WARN[0000] no retention policies set for destination, keeping everything
```

Retention policies specifies what backups (and snapshots) you want to
retain (keep). Pruning means removing every backup/snapshot that is not
retained by the retention policy. Pruning is automatically performed
after every backup (except if the `--no-prune` flag is specified on the
command line) and can be manually performed with `uback prune`.

For backups, the default policy is to retain everything.

For bookmarks, the unique policy (not configurable) is to only retain
the ones that are present in `StateFile`, which just means the latest
successful backup present on each destination.

For archives, the default policy is to retain nothing, except the
snapshots that are present in `StateFile` and not covered by a bookmark.

Archives retention policies are specified on the source, whereas backups
retention policies are specified on the destination.

We will not discuss archives retention and pruning ; it works exactly the
same way as backups.

A retention policy consists of three pieces of information :

* An interval
* A count
* An optional `full` option

The syntax is `interval=count[:full]`, for example `weekly=13:full` or
`daily=7`. The syntax for the interval is the same than the syntax for
`full-interval`, described earlier in the `Incremental Backups` section.

The `count` most recents backups are retained, as long as they are
separated by (at least) `interval`. If the `full` option is specified,
only full backups are considered. Then, all backups required to restore
a retained backups are marked as retained, bottoming up to a full
backup. Eventually, all backups not marked as retained are pruned.

A notable exception to this process is orphan incremental backups,
i.e. incremental backups that do not eventually bottom up to an existing
full backup. Those are defective (because they cannot be restored)
and are always pruned.

For example, let’s say we have those backups :

```
2020-12-31 (full)
2021-01-01 (full)
2021-01-02 (incremental, from 2021-01-01)
2021-01-03 (incremental, from 2021-01-02)
2021-01-04 (incremental, from 2021-01-03)
2021-01-05 (incremental, from 2021-01-04)
2021-01-06 (incremental, from 2021-01-05)
2021-01-07 (full)
2021-01-08 (incremental, from 2021-01-07)
2021-01-09 (incremental, from 2019-12-01)
2021-01-10 (incremental, from 2021-01-09)
```

First, let’s note that no matter the retention policy (except the
default retention policy which keeps all backups, no exception) the last
two backups in this list will always be pruned, because they are orphan
incremental backups. Let’s rid them out of the picture and consider
the list without orphan backups :

```
2020-12-31 (full)
2021-01-01 (full)
2021-01-02 (incremental, from 2021-01-01)
2021-01-03 (incremental, from 2021-01-02)
2021-01-04 (incremental, from 2021-01-03)
2021-01-05 (incremental, from 2021-01-04)
2021-01-06 (incremental, from 2021-01-05)
2021-01-07 (full)
2021-01-08 (incremental, from 2021-01-07)
```

Let’s go through some examples :

1. The `daily=1` retention policy will keep the last two backups : it will
mark the last backup as retained, and then keep the one before the last
as a dependency.

2. The `daily=1:full` retention policy will only keep the `2021-01-07`
backup.

3. The `daily=3:full` retention policy will keep the three full backups,
namely `2021-01-07`, `2021-01-01` and `2020-12-31`.

4. The `daily=3` retention policy will keep everything except the
first backup.

5. The `5d=2` retention policy will keep `2021-01-08` and `2021-01-03`
and their dependencies (`2021-01-07`, `2021-01-02` and `2021-01-01`).

6. If multiple retention policies are given, all retention policies are
evaluated independently, and the backups that are retained are the one
that are retained by at least policy (in other words, only the backups
that are retained by no retention policy are pruned).

Let’s illustrate all that by creating a fake repository of backups
mirroring our list above, and by appling the retention policies 3. and
5. :

```
$ mkdir -p /tmp/my-backups/test-retention
$ touch /tmp/my-backups/test-retention/20201231T000000.000-full.ubkp
$ touch /tmp/my-backups/test-retention/20210101T000000.000-full.ubkp
$ touch /tmp/my-backups/test-retention/20210102T000000.000-from-20210101T000000.000.ubkp
$ touch /tmp/my-backups/test-retention/20210103T000000.000-from-20210102T000000.000.ubkp
$ touch /tmp/my-backups/test-retention/20210104T000000.000-from-20210103T000000.000.ubkp
$ touch /tmp/my-backups/test-retention/20210105T000000.000-from-20210104T000000.000.ubkp
$ touch /tmp/my-backups/test-retention/20210106T000000.000-from-20210105T000000.000.ubkp
$ touch /tmp/my-backups/test-retention/20210107T000000.000-full.ubkp
$ touch /tmp/my-backups/test-retention/20210108T000000.000-from-20210107T000000.000.ubkp
$ uback prune backups -n type=fs,path=/tmp/my-backups/test-retention,@retention-policy=daily=3:full,@retention-policy=5d=2
20210106T000000.000
20210105T000000.000
20210104T000000.000
$ uback prune backups type=fs,path=/tmp/my-backups/test-retention,@retention-policy=daily=3:full,@retention-policy=5d=2
20210106T000000.000
20210105T000000.000
20210104T000000.000
$ uback list backups type=fs,path=/tmp/my-backups/test-retention,@retention-policy=daily=3:full,@retention-policy=5d=2
20201231T000000.000 (full)
20210101T000000.000 (full)
20210102T000000.000 (base: 20210101T000000.000)
20210103T000000.000 (base: 20210102T000000.000)
20210107T000000.000 (full)
20210108T000000.000 (base: 20210107T000000.000)
```

Two things to note here :

* The retention policies are specified as a destination option for
backups. The name of the option is `@retention-policy` ; the `@` indicates
that the option can have multiple values (here, a destination can have
multiple retention policies). The `@` symbol must be specified even if
you have only one retention policy.

* `prune -n` allows you to see what a retention policy would prune
without actually removing the backups, for testing purposes. Once you
are sastified with a retention policy, you can add it to your preset or
your backup command for automatic pruning on every backup.
