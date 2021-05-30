load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/src-command"

@test "command source" {
	export PATH="$PATH:$BATS_TEST_DIRNAME"
	mkdir -p "$TEST_TMPDIR"

	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	source=type=command,command=uback-tar-src,path="$TEST_TMPDIR/source",key-file="$TEST_TMPDIR/backup.pub",state-file="$TEST_TMPDIR/state.json",snapshots-path="$TEST_TMPDIR/snapshots",full-interval=weekly,@extra-args=--exclude=./c,@extra-args=--exclude=./d
	dest=id=test,type=fs,path="$TEST_TMPDIR/backups",@retention-policy=daily=3,key-file="$TEST_TMPDIR/backup.key"

	mkdir -p "$TEST_TMPDIR/source/d"

	echo "av1" > "$TEST_TMPDIR/source/a"
	echo "c" > "$TEST_TMPDIR/source/c"
	echo "e" > "$TEST_TMPDIR/source/d/e"
	$UBACK backup "$source" "$dest"
	$UBACK restore -d "$TEST_TMPDIR/restore" "$dest"
	assert_equal "$(cat "$TEST_TMPDIR"/restore/*/a)" "av1"
	[ ! -e "$TEST_TMPDIR"/restore/*/c ]
	[ ! -e "$TEST_TMPDIR"/restore/*/d ]
	rm -rf "$TEST_TMPDIR"/restore/*
	b1=$(cd "$TEST_TMPDIR"/backups; ls | sort | tail -n 1)
	sleep 0.001

	echo "bv1" > "$TEST_TMPDIR/source/b"
	$UBACK backup "$source" "$dest"
	$UBACK restore -d "$TEST_TMPDIR/restore" "$dest"
	assert_equal "$(cat "$TEST_TMPDIR"/restore/*/b)" "bv1"
	[ ! -e "$TEST_TMPDIR"/restore/*/c ]
	[ ! -e "$TEST_TMPDIR"/restore/*/d ]
	rm -rf "$TEST_TMPDIR"/restore/*
	b2=$(cd "$TEST_TMPDIR"/backups; ls | sort | tail -n 1)
	sleep 0.001

	echo "av2" > "$TEST_TMPDIR/source/a"
	$UBACK backup "$source" "$dest"
	$UBACK restore -d "$TEST_TMPDIR/restore" "$dest"
	assert_equal "$(cat "$TEST_TMPDIR"/restore/*/a)" "av2"
	assert_equal "$(cat "$TEST_TMPDIR"/restore/*/b)" "bv1"
	[ ! -e "$TEST_TMPDIR"/restore/*/c ]
	[ ! -e "$TEST_TMPDIR"/restore/*/d ]
	rm -rf "$TEST_TMPDIR"/restore/*
	b3=$(cd "$TEST_TMPDIR"/backups; ls | sort | tail -n 1)

	# Check that incremental backups are actually incremental
	<"$TEST_TMPDIR/backups/$b2" $UBACK container extract -k "$TEST_TMPDIR/backup.key" | tar -C "$TEST_TMPDIR/restore" -x
	[ ! -e "$TEST_TMPDIR"/restore/a" ]
	assert_equal "$(cat "$TEST_TMPDIR"/restore/b)" "bv1"
	rm -rf "$TEST_TMPDIR"/restore/*

	<"$TEST_TMPDIR/backups/$b3" $UBACK container extract -k "$TEST_TMPDIR/backup.key" | tar -C "$TEST_TMPDIR/restore" -x
	assert_equal "$(cat "$TEST_TMPDIR"/restore/a)" "av2"
	[ ! -e "$TEST_TMPDIR"/restore/b" ]
	rm -rf "$TEST_TMPDIR"/restore/*
}
