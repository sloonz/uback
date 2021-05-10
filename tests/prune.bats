load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/prune"

setup() {
	mkdir -p "$TEST_TMPDIR"
}

@test "manual pruning" {
	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	source=type=tar,path="$TEST_TMPDIR/source",key-file="$TEST_TMPDIR/backup.pub",state-file="$TEST_TMPDIR/manual/state.json",snapshots-path="$TEST_TMPDIR/manual/snapshots",full-interval=weekly
	dest=id=test,type=fs,path="$TEST_TMPDIR/manual/backups",@retention-policy=daily=3

	mkdir -p "$TEST_TMPDIR/manual/snapshots"
	mkdir -p "$TEST_TMPDIR/manual/backups"
	
	touch "$TEST_TMPDIR/manual/snapshots/20210101T000000.000"
	touch "$TEST_TMPDIR/manual/snapshots/20210102T000000.000"
	touch "$TEST_TMPDIR/manual/snapshots/20210103T000000.000"
	touch "$TEST_TMPDIR/manual/snapshots/20210104T000000.000"
	touch "$TEST_TMPDIR/manual/snapshots/20210105T000000.000"
	touch "$TEST_TMPDIR/manual/snapshots/20210106T000000.000"
	touch "$TEST_TMPDIR/manual/backups/20210101T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/manual/backups/20210102T000000.000-from-20210101T000000.000.ubkp"
	touch "$TEST_TMPDIR/manual/backups/20210103T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/manual/backups/20210104T000000.000-from-20210103T000000.000.ubkp"
	touch "$TEST_TMPDIR/manual/backups/20210105T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/manual/backups/20210106T000000.000-from-20210105T000000.000.ubkp"
	echo '{"test":"20210106T000000.000"}' > "$TEST_TMPDIR/manual/state.json"

	assert_equal "$($UBACK list snapshots "$source" | wc -l)" 6
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 6

	$UBACK prune snapshots "$source"
	$UBACK prune backups "$dest"

	assert_equal "$($UBACK list snapshots "$source" | wc -l)" 1
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 4

	[ -f "$TEST_TMPDIR/manual/snapshots/20210106T000000.000" ]
	[ -f "$TEST_TMPDIR/manual/backups/20210106T000000.000-from-20210105T000000.000.ubkp" ]
	[ -f "$TEST_TMPDIR/manual/backups/20210105T000000.000-full.ubkp" ]
	[ -f "$TEST_TMPDIR/manual/backups/20210104T000000.000-from-20210103T000000.000.ubkp" ]
	[ -f "$TEST_TMPDIR/manual/backups/20210103T000000.000-full.ubkp" ]
}

@test "automatic pruning" {
	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	source=type=tar,path="$TEST_TMPDIR/source",key-file="$TEST_TMPDIR/backup.pub",state-file="$TEST_TMPDIR/automatic/state.json",snapshots-path="$TEST_TMPDIR/automatic/snapshots",full-interval=weekly
	dest=id=test,type=fs,path="$TEST_TMPDIR/automatic/backups",@retention-policy=daily=3

	mkdir -p "$TEST_TMPDIR/automatic/snapshots"
	mkdir -p "$TEST_TMPDIR/automatic/backups"
	mkdir -p "$TEST_TMPDIR/source"
	
	touch "$TEST_TMPDIR/automatic/snapshots/20210101T000000.000"
	touch "$TEST_TMPDIR/automatic/snapshots/20210102T000000.000"
	touch "$TEST_TMPDIR/automatic/snapshots/20210103T000000.000"
	touch "$TEST_TMPDIR/automatic/snapshots/20210104T000000.000"
	touch "$TEST_TMPDIR/automatic/snapshots/20210105T000000.000"
	touch "$TEST_TMPDIR/automatic/snapshots/20210106T000000.000"
	touch "$TEST_TMPDIR/automatic/backups/20210101T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/automatic/backups/20210102T000000.000-from-20210101T000000.000.ubkp"
	touch "$TEST_TMPDIR/automatic/backups/20210103T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/automatic/backups/20210104T000000.000-from-20210103T000000.000.ubkp"
	touch "$TEST_TMPDIR/automatic/backups/20210105T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/automatic/backups/20210106T000000.000-from-20210105T000000.000.ubkp"
	echo '{"test":"20210106T000000.000"}' > "$TEST_TMPDIR/automatic/state.json"

	assert_equal "$($UBACK list snapshots "$source" | wc -l)" 6
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 6

	$UBACK backup "$source" "$dest"

	assert_equal "$($UBACK list snapshots "$source" | wc -l)" 1
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 3

	[ -f "$TEST_TMPDIR/automatic/backups/20210106T000000.000-from-20210105T000000.000.ubkp" ]
	[ -f "$TEST_TMPDIR/automatic/backups/20210105T000000.000-full.ubkp" ]
}

@test "disabled automatic pruning" {
	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	source=type=tar,path="$TEST_TMPDIR/source",key-file="$TEST_TMPDIR/backup.pub",state-file="$TEST_TMPDIR/no-prune/state.json",snapshots-path="$TEST_TMPDIR/no-prune/snapshots",full-interval=weekly
	dest=id=test,type=fs,path="$TEST_TMPDIR/no-prune/backups",@retention-policy=daily=3

	mkdir -p "$TEST_TMPDIR/no-prune/snapshots"
	mkdir -p "$TEST_TMPDIR/no-prune/backups"
	mkdir -p "$TEST_TMPDIR/source"
	
	touch "$TEST_TMPDIR/no-prune/snapshots/20210101T000000.000"
	touch "$TEST_TMPDIR/no-prune/snapshots/20210102T000000.000"
	touch "$TEST_TMPDIR/no-prune/snapshots/20210103T000000.000"
	touch "$TEST_TMPDIR/no-prune/snapshots/20210104T000000.000"
	touch "$TEST_TMPDIR/no-prune/snapshots/20210105T000000.000"
	touch "$TEST_TMPDIR/no-prune/snapshots/20210106T000000.000"
	touch "$TEST_TMPDIR/no-prune/backups/20210101T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/no-prune/backups/20210102T000000.000-from-20210101T000000.000.ubkp"
	touch "$TEST_TMPDIR/no-prune/backups/20210103T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/no-prune/backups/20210104T000000.000-from-20210103T000000.000.ubkp"
	touch "$TEST_TMPDIR/no-prune/backups/20210105T000000.000-full.ubkp"
	touch "$TEST_TMPDIR/no-prune/backups/20210106T000000.000-from-20210105T000000.000.ubkp"
	echo '{"test":"20210106T000000.000"}' > "$TEST_TMPDIR/no-prune/state.json"

	assert_equal "$($UBACK list snapshots "$source" | wc -l)" 6
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 6

	$UBACK backup -n "$source" "$dest"

	assert_equal "$($UBACK list snapshots "$source" | wc -l)" 7
	assert_equal "$($UBACK list backups "$dest" | wc -l)" 7
}
