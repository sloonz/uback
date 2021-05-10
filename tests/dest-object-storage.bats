load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/dest-object-storage"

local container

setup() {
	mkdir -p "$TEST_TMPDIR"
	docker network create --driver=bridge uback-minio-test-bridge
	container="$(docker container create --network uback-minio-test-bridge -h minio -p 9000:9000 minio/minio server /data)"
	docker container start "$container"
	for i in $(seq 1 300); do
		if docker run --network uback-minio-test-bridge --entrypoint=/bin/sh -i minio/mc -c "mc alias set minio http://minio:9000 minioadmin minioadmin && mc mb minio/testbucket" ; then
			break
		else
			refute [ "$i" = 300 ]
			sleep 0.1
		fi
	done
}

teardown() {
	docker container rm -f "$container"
	docker network rm uback-minio-test-bridge
}

@test "object-storage destination" {
	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	source=type=tar,path="$TEST_TMPDIR/source",key-file="$TEST_TMPDIR/backup.pub",state-file="$TEST_TMPDIR/state.json",snapshots-path="$TEST_TMPDIR/snapshots",full-interval=weekly
	dest=id=test,type=object-storage,@retention-policy=daily=3,key-file="$TEST_TMPDIR/backup.key",url=http://minioadmin:minioadmin@localhost:9000/testbucket,prefix=/test

	mkdir -p "$TEST_TMPDIR/restore"
	mkdir -p "$TEST_TMPDIR/source"
	echo "hello" > "$TEST_TMPDIR/source/a"

	# Full 1
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 0
	$UBACK backup -n -f "$source" "$dest"
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 1
	sleep 0.01

	# Full 2
	$UBACK backup -n -f "$source" "$dest"
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 2
	sleep 0.01

	# Incremental
	echo "world" > "$TEST_TMPDIR/source/b"
	$UBACK backup -n "$source" "$dest"
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 3

	# Prune (remove full 1)
	$UBACK prune backups "$dest"
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 2

	# Restore full 2 + incremental
	$UBACK restore -d "$TEST_TMPDIR/restore" "$dest"
	assert_equal "$(cat "$TEST_TMPDIR"/restore/*/a)" "hello"
	assert_equal "$(cat "$TEST_TMPDIR"/restore/*/b)" "world"
}
