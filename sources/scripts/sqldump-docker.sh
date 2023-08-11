#!/bin/sh

set -xe

datadir=$(realpath "$(dirname "$0")")
version=$(perl -ne 'print $1 if /^server_version = (\d+\.\d+\.\d+)/' "$datadir/xtrabackup_info")
container=$(docker container run --rm -u "$UID" -v "$datadir":/var/lib/mysql -di mariadb:"$version" mariadbd --skip-grant-tables)

for i in $(seq 1 300) ; do
	if ! docker exec -i "$container" mariadb -h127.0.0.1 -e "SELECT VERSION()" > /dev/null; then
		sleep 0.1
	else
		break
	fi
done

docker exec -i "$container" mariadb-dump "$@"
docker container stop "$container" > /dev/null
