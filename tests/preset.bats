load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/preset"

@test "presets" {
	mkdir -p "$TEST_TMPDIR"

	$UBACK -p "$TEST_TMPDIR/presets" preset set tar-src "@Command=sudo,@Command=tar"
	assert_equal "$($UBACK -p "$TEST_TMPDIR/presets" preset list -v)" "tar-src [[@Command sudo] [@Command tar]]"
	$UBACK -p "$TEST_TMPDIR/presets" preset remove tar-src

	$UBACK -p "$TEST_TMPDIR/presets" preset set tar-src "@Command=sudo"
	assert_equal "$($UBACK -p "$TEST_TMPDIR/presets" preset list -v)" "tar-src [[@Command sudo]]"
	$UBACK -p "$TEST_TMPDIR/presets" preset set tar-src "@Command=tar"
	assert_equal "$($UBACK -p "$TEST_TMPDIR/presets" preset list -v)" "tar-src [[@Command sudo] [@Command tar]]"
	$UBACK -p "$TEST_TMPDIR/presets" preset remove tar-src

	$UBACK -p "$TEST_TMPDIR/presets" preset set escape-path escaped-path='{{.Path|clean|replace "/" "-"|trimSuffix "-"}}'
	$UBACK -p "$TEST_TMPDIR/presets" preset set src state-file='/var/lib/uback/state/{{.EscapedPath}}.json' key-file='/etc/uback/backup.pub'
	$UBACK -p "$TEST_TMPDIR/presets" preset set tar-src type=tar preset=escape-path preset=src
	config=$($UBACK -p "$TEST_TMPDIR/presets" preset eval path=/etc,preset=tar-src)
	sorted_config=$(printf "%s" "$config" | sort)
	expected_config=$(cat <<eof
EscapedPath: -etc
KeyFile: /etc/uback/backup.pub
Path: /etc
StateFile: /var/lib/uback/state/-etc.json
Type: tar
eof)
	assert_equal "$sorted_config" "$expected_config"
}
