#!/bin/sh

set -xe

datadir=$(realpath "$(dirname "$0")")
container=$(docker container create -v "$datadir":/var/lib/mysql mariadb:latest mysqld --skip-grant-tables)

docker container start "$container" > /dev/null

for i in $(seq 1 300) ; do
	if ! docker exec -i "$container" mysql -h127.0.0.1 -e "SELECT VERSION()" > /dev/null; then
		sleep 0.1
	else
		break
	fi
done

docker exec -i "$container" mysqldump "$@"
docker container rm -f "$container" > /dev/null
