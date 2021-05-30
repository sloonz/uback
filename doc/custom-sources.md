# Custom Sources

Custom sources are sources that are not built-in into `uback`, but
implemented by an external command, typically a script, under the control
of the user.

## Custom Sources for Users

When creating a backup, you give `command` as the source type, and set the
`@SourceCommand` option to the custom source command. It can be either
a full path or relative path if the custom source command in in the PATH.

```shell
$ uback backup type=command,@source-command=uback-tar-src,path=/etc,snapshots-path=/var/lib/uback/custom-tar-snapshots/ $DEST_OPTIONS
```

When restoring a backup, on the other hand, the custom command MUST be
in the PATH, since the name of the custom source command will be extracted
from the backup file header.

## Custom Sources for Implementors

The first argument passed to the command is the kind of command: `source`
for a source, and `destination` for a destination. In the context of
this document, the first argument will always be `source`.

`uback` passes the requested operation and arguments as the next arguments
on the command line, and options passed to the source as environment
variable. For example, the `SnapshotsPath` will be translated into the
`UBACK_OPT_SNAPSHOTS_PATH` environment variable. If the option is a
slice option (for example `@AdditionalArguments`), it will be passed into
`UBACK_SOPT_ADDITIONAL_ARGUMENTS`, and the value will be a JSON-serialized
string array (the `S` in `SOPT` stands for "slice").

A custom source command should follow the usual external process
conventions : use stdout for normal output that will be consumed by
`uback`, use stdin for normal input that will be given by `uback`,
use stderr to print free-format messages destined to the end user. An
exit code of 0 indicates a successful operation, whereas a non-zero one
indicates a failure.

A source must implement those operations, which will be described in
the next sections :

* type
* list-snapshots
* remove-snapshot
* create-backup
* restore-backup

## Operations

### type

This operation takes no argument, and should just prints the type of the
source, which will be used in the restoration process to determine what
source will be responsible with restoring the backup.

It should be `command:script-name`, where `script-name` is the normal name
of the custom source command. If the `command:` prefix is not provided, it
is automatically added (after emitting a warning) ; you can prevent that
behavior by adding a `:` prefix (the `:` prefix will then be stripped).

It is also a good place to validate the options.

### list-snapshots

This operation takes no argument, and should print all snapshots usable
for creating incremental backups, one per line.

`list-snapshots` is allowed to return invalid snapshots names as long
as they start with a `.` or a `_`. Those entries will be ignored.

### remove-snapshot

This operation takes one argument, and should remove the snapshot
identified by the argument.

### create-backup

This operation takes one optional argument, the base snapshot. If
present, the source must try to create an incremental backup from the
given snapshot. If not present, the source must create a new full backup.

The source must first prints the name of the backup in a single line
(`(snapshot)-full` for a full backup, `(snapshot)-from-(baseSnapshot)`
for an incremental backup) and then just stream the backup data to stdout.

### restore-backup

Note that as a special cases, the options are not validated by the `type`
operation before calling this operation: in a backup restoration, this
operation is directly called.

The first argument of the operation is the target directory, where to
restore the backup.

The second argument of the operation is the backup snapshot.

The third argument of the operation is optional ; it is the backup base
snapshot if the backup is an incremental one.

If the backup is an incremental one, it is guaranteed that
`restore-backup` has been called on the base just before.

Backup data is passed to the custom source via stdin. No output is
expected from the custom source.

## Example

As an example, you can look at the [uback-tar-src](../tests/uback-tar-src)
test script, which reimplement the tar source as a bash script.
