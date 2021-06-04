load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/container"

@test "container" {
	mkdir -p "$TEST_TMPDIR"
	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	echo hello | $UBACK container create -k "$TEST_TMPDIR/backup.pub" test > "$TEST_TMPDIR/test.ubkp"
	assert_equal "test" "$($UBACK container type "$TEST_TMPDIR/test.ubkp")"
	assert_equal "hello" "$($UBACK container extract -k "$TEST_TMPDIR/backup.key" < "$TEST_TMPDIR/test.ubkp")"
}
