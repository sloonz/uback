# Custom Destinations

Custom destinations are destintations that are not build-in into `uback`,
but implemented by an external command, typicall a script, under the
control of the user.

## Custom Destinations for Users

When creating a backup, you give `command` as the destination type,
and set the `Command` option to the custom source command. It can be
either a full path or relative path if the custom destination command
in in the PATH.

```shell
$ uback backup $SOURCE_OPTIONS id=test,type=custom,command=uback-fs-dest
```

## Custom Destinations for Implementors

The first argument passed to the command is the kind of command: `source`
for a source, and `destination` for a destination. In the context of
this document, the first argument will always be `destination`.

`uback` passes the requested operation and arguments as the next arguments
on the command line, and options passed to the destination as environment
variable. For example, the `SnapshotsPath` will be translated into the
`UBACK_OPT_SNAPSHOTS_PATH` environment variable. If the option is a
slice option (for example `@AdditionalArguments`), it will be passed into
`UBACK_SOPT_ADDITIONAL_ARGUMENTS`, and the value will be a JSON-serialized
string array (the `S` in `SOPT` stands for "slice").

A custom destination command should follow the usual external process
conventions : use stdout for normal output that will be consumed by
`uback`, use stdin for normal input that will be given by `uback`,
use stderr to print free-format messages destined to the end user. An
exit code of 0 indicates a successful operation, whereas a non-zero one
indicates a failure.

A destination must implement those operations, which will be described in
the next sections :

* validate-options
* list-backups
* remove-backup
* send-backup
* receive-backup

## Operations

### validate-options

This operation takes no argument ; it is called before any other operation
and should be used to validate options.

### list-backups

This operation takes no argument, and should print all backups currently
present on the destination, one per line. You can either
give a backup name (`20210102T000000.000-full`) or a filename
(`20210102T000000.000-full.ubkp`).

For convenience purposes, lines starting with a `.` or `_` are ignored.

### remove-backup

This operation takes one argument, the full
name of a backup (`20210102T000000.000-full` or
`20210102T000000.000-from-20210101T000000.000`), and should remove the
backup on the destination.

### send-backup

This operation takes one argument, the full name of a backup. `uback`
will provide the backup to the command standard input ; the command
should store it.

### receive-backup

This operation takes one argument, the full name of a backup. The command
should output the backup on its standard output.

## Example

As an example, you can look at the [uback-fs-test](../tests/uback-fs-test)
test script, which reimplement the fs destination as a bash script.
