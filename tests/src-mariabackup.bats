load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/src-mariabackup"

local container

setup() {
	if [ "$SKIP_MARIADB_TESTS" = "1" ] ; then
		skip
	fi

	mkdir -p "$TEST_TMPDIR"

	container=$(docker container create -v "$TEST_TMPDIR/snapshots:$TEST_TMPDIR/snapshots" -e MARIADB_ROOT_PASSWORD=test mariadb:latest)
	docker container start "$container"
	docker exec "$container" chmod 777 "$TEST_TMPDIR/snapshots"

	# wait for mysql
	for i in $(seq 1 300) ; do
		if ! docker exec -i "$container" mysql -h127.0.0.1 -uroot -ptest -e "SELECT VERSION()" ; then
			sleep 0.1
		else
			break
		fi
	done

	docker exec -i "$container" mysql -uroot -ptest -e "SELECT VERSION()"
}

teardown() {
	if [ "$SKIP_MARIADB_TESTS" = "1" ] ; then
		skip
	fi

	docker exec -i "$container" bash -c "rm -rf \"$TEST_TMPDIR\"/snapshots/*"
	docker container rm -f "$container"
}

@test "mariabackup source" {
	# early fail if test dependencies are not present
	which mysql
	which mysqldump
	which mbstream
	which mariabackup

	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	source=type=mariabackup,key-file="$TEST_TMPDIR/backup.pub",state-file="$TEST_TMPDIR/state.json",snapshots-path="$TEST_TMPDIR/snapshots",command="docker exec -i $container mariabackup -uroot -ptest",full-interval=weekly
	dest=id=test,type=fs,path="$TEST_TMPDIR/backups",@retention-policy=daily=3,key-file="$TEST_TMPDIR/backup.key"

	docker exec -i "$container" mysql -uroot -ptest <<eof
create database ubkptest;
eof

	docker exec -i "$container" mysql -uroot -ptest ubkptest <<eof
create table test(a int);
insert into test values (1);
eof
	$UBACK backup -n "$source" "$dest"
	sleep 0.01

	docker exec -i "$container" mysql -uroot -ptest ubkptest <<eof
insert into test values (2);
insert into test values (3);
eof
	$UBACK backup -n "$source" "$dest"
	sleep 0.01

	docker exec -i "$container" mysql -uroot -ptest ubkptest <<eof
update test set a=4 where a=1;
delete from test where a=2;
eof
	$UBACK backup -n "$source" "$dest"

	$UBACK restore -d "$TEST_TMPDIR/restore" "$dest"

	restore_path=$(echo "$TEST_TMPDIR"/restore/*/)
	cp -a "$restore_path" "$TEST_TMPDIR"/restore2
	"$TEST_TMPDIR"/restore2/sqldump-docker.sh ubkptest > "$TEST_TMPDIR"/restore/sqldump-docker.sql
	assert grep -Fq 'INSERT INTO `test` VALUES (4),(3);' "$TEST_TMPDIR"/restore/sqldump-docker.sql
	docker container run -v "$TEST_TMPDIR"/restore2:/var/lib/mysql mariadb:latest bash -c "rm -rf /var/lib/mysql/*"

	"$TEST_TMPDIR"/restore/*/sqldump-local.sh ubkptest > "$TEST_TMPDIR"/restore/sqldump-local.sql
	assert grep -Fq 'INSERT INTO `test` VALUES (4),(3);' "$TEST_TMPDIR"/restore/sqldump-local.sql
}
