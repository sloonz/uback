# mariabackup Source

Backup a MariaDB instance using mariabackup. Supports incremental backups.

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

Optional, defaults: `[mariadb-backup]`

Caveat emptor : this may be tricky when used in conjonction with the
`User` or `Password` options.

The `User` and `Password` options create a temporary file and pass
it to `mariadb-backup` via the `--defaults-file` option. However,
`mariadb-backup` requires the `--defaults-file` option to be the first one
passed in the command line. So while you can perfectly combine the `User`
or `Password` option with `@Command` if your intention is to prepend
stuff to `mariabackup` (for example to change the base command to `sudo -u
dbuser mariabackup`), you cannot do so to append stuff (extra arguments,
like `--databases-exclude`) if you have `User` or `Password` set.

If you run into this issue, the recommended solution is to simply set-up
Unix Authentication for the user that will run `mariabackup` so you can
authenticate without password the user with `@Command=mariadb-backup
-ubackupuser`.

### VersionCheck

Optional, defaults: `true`

When doing an incremental backup, check that base backup server version
and current server version are the same. If they are different, force
a full backup.

### @MariadbCommand

Optional, defaults: `[mariadb]`

Same remarks as `Command` apply. This is only used for server version
check.

### UseDocker (restoration only)

Optional, defaults: `true`

During the restoration process, `mariadb-backup --prepare` must be run,
with the same version than the server that created the backup (althought
it tend to work even with version mismatch). If this option is set
to true, run the command in a docker process with the correct mariadb
version. If false, uses `@Command` for the restoration process.
