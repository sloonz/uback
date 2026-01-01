#!/bin/sh

set -xe

datadir=$(realpath "$(dirname "$0")")
if [ -f "$datadir/mariadb_backup_info" ] ; then
	version=$(perl -ne 'print $1 if /^server_version = (\d+\.\d+\.\d+)/' "$datadir/mariadb_backup_info")
else
	version=$(perl -ne 'print $1 if /^server_version = (\d+\.\d+\.\d+)/' "$datadir/xtrabackup_info")
fi
container=$(podman run --rm -v "$datadir":/var/lib/mysql -di docker.io/library/mariadb:"$version" mariadbd --skip-grant-tables)

for i in $(seq 1 300) ; do
	if ! podman exec -i "$container" mariadb -h127.0.0.1 -e "SELECT VERSION()" > /dev/null; then
		sleep 0.1
	else
		break
	fi
done

podman exec -i "$container" mariadb-dump "$@"
podman stop "$container" > /dev/null
