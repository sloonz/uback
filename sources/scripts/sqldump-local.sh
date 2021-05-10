#!/bin/sh

set -xe

datadir=$(realpath "$(dirname "$0")")
socket=$(realpath "$datadir/../mysqld.sock")

mysqld --skip-grant-tables --skip-networking --datadir="$datadir" --socket="$socket" &
pid=$!

for i in $(seq 1 300) ; do
	if ! mysql --socket="$socket" -e "SELECT VERSION()" > /dev/null; then
		sleep 0.1
	else
		break
	fi
done

mysqldump --socket="$socket" "$@"
kill "$pid"
wait
