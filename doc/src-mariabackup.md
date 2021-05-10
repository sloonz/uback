# mariabackup Source

Backup a MariaDB instance using mariabackup. Supports incremental backups.

## Limitations

As stated in the `mariabackup` documentation, `mariabackup` version and
the mariadb server version should be the same when creating a backup.

While `mariabackup` won’t complain when creating an incremental
backup based on a full backup made by a previous version, nowhere in the
documentation it is explicitly stated that it is a valid operation and
it makes the restoration process much more complicated, and is therefore
unsupported by `uback`. The simplest way to avoid any trouble related to
this is to remove the contents of `SnapshotsPath` every time you upgrade
mariadb, so a full backup will be forced at next backup.

## Notes on restoration

Contrarily to `mariadb-dump`, the result of a restoration is a `mariadb`
data dir (that you can directly restore into `/var/lib/mysql`), not a
SQL file.

`uback` will create two helpers scripts in the restored directory,
`dumpsql-local.sh` and `dumpsql-docker.sh`, that will start a temporary
(warning: unsecured) MariaDB server and will run `mariadb-dump` on it,
for example :

```
$ /path/to/restored/data/dumpsql-local.sh --all-databases > backup.sql
```

`dumpsql-local.sh` will use a local system-wide installation of MariaDB,
whereas `dumpsq-local.sh` will spawn a docker container.

## Options

### SnapshotsPath

Optional, but required for incremental backups.

### User

Optional.

Note that contrary to the `mariadb` command line client, an empty user
does not defaults to the current Unix user, but simply to the empty user.

### Password

Optional, defaults to empty (which is perfectly fine if [Unix
Authentication](https://mariadb.com/kb/en/authentication-plugin-unix-socket/)
is properly configured).

### @Command

Optional, defaults: `[mariabackup]`

Caveat emptor : this may be tricky when used in conjonction with the
`User` or `Password` options.

The `User` and `Password` options create a temporary file and pass it to
`mariabackup` via the `--defaults-file` option. However, `mariabackup`
requires the `--defaults-file` option to be the first one passed in the
command line. So while you can perfectly combine the `User` or
`Password` option with `@Command` if your intention is to prepend stuff
to `mariabackup` (for example to change the base command to `sudo -u
dbuser mariabackup`), you cannot do so to append stuff (extra arguments,
like `--databases-exclude`) if you have `User` or `Password` set.

If you run into this issue, the recommended solution is to simply set-up
Unix Authentication for the user that will run `mariabackup` so you can
authenticate without password the user with `@Command=-ubackupuser`.
