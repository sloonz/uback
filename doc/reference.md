# Reference

## Options, Presets and Templates

[Go Template Syntax Reference](https://golang.org/pkg/text/template/)

[Spig Functions Reference](https://masterminds.github.io/sprig/)

Both presets and command-line options are modeled as a list of key-value
pairs. Keys are converted in PascalCase when read from the command
line. Keys starting with `@` means that all values are used, whereas
for other keys only the last one is used.

Options are first split is key-value pairs separated by a comma
(","). Commas can be escaped by a backslash ("\"). Backaslashes can be
escaped by a backslash.

Values in the list are evaluated as a Go Template with Sprig
Functions. All previously evaluated key-values pairs are accessible in
the template (use `{{index . "@Option"}}` to access an option whose key
starts with an "@" ; the associated value will be a string slice). Presets
are (recursively) substituted by the key-value pairs they designate.

## Time Intervals and Retention Policies

A time interval is used to specify the maximum time between two full
backups or the minimum time between two retained backups in the retention
policies. It is a positive number, optionally followed by one of those
one-letter units :

* `h` for one hour (60 seconds)
* `d` for one day (24 hours)
* `w` for one week (7 days)
* `m` for one month (30 days)
* `y` for one year (365 days)

If no unit is given, the default unit is one second.

Those aliases are also available :

* `hourly` for `1h`
* `daily` for `1d`
* `weekly` for `1w`
* `monthly` for `1m`
* `yearly` for `1y`

A retention policy consists of an interval, a count, and an optional
`full` option. It is formatted like this : `interval=count[:full]`.

When multiple retention policies are given, an item is retained if it
is retained by at least one retention policy.

A retention policy will mark as retained the `count` most recent items,
under the constraints that retained items must be separated by at least
`interval`. If the `full` option is given, incremental items are not
considered. After this first list of retained items is built, it is
enhanced by the (recursive) dependencies of the retained items. At the
end of this process, all items not marked as retained are prunable.

## Common Source Options

### StateFile

The `StateFile` keeps tracks of the last backup present on every
destination where this source has been backed up to.

### @RetentionPolicy

Specify the list of retention policies applied to snapshots of this
source. If none is given, the default retention policy is to retain
nothing.

Snapshots referenced in `StateFile` are always retained no mattter what
the retention policies are.

### FullInterval

Maximum time between two full backups : the the most recent full backup
is older than the interval, then force the creation of a new full backup
even if an incremental backup could have been created.

### Key / KeyFile

Gives the public key for backup file encryption, either as a PEM file
(`KeyFile`) or as an inline DER base64-encoded key (which is the base64
content of the PEM file).

Although this is discouraged, you can also provide the private key and
`uback` will automatically derive the public key from it, allowing you
to use the private key in a similar way to a symmetric key.

## Common Destination Options

### ID

An identifier to identify the destination in the `StateFile` file. It
must be unique for a source-destination pair, but the same destination
can share the same identifier across different sources.

### @RetentionPolicy

Specify the list of retention policies applied to backups of this
destination. If none is given, the default retion policy is to retain
everything.

Orphans incremental backups (that do not eventually bottom out to an
existing full backup) are always pruned, except in the case of the
default policy.

### Key / KeyFile

Gives the private key for backup file decryption, either as a PEM file
(`KeyFile`) or as an inline DER base64-encoded key (which is the base64
content of the PEM file).

This should option should NEVER be provided during normal backup
operations, only during restoration of a backup.
