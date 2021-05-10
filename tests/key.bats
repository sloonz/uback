load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/key"

setup() {
	mkdir -p "$TEST_TMPDIR"
}

@test "key generation" {
	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	assert_equal "$(openssl pkey -pubout < "$TEST_TMPDIR/backup.key")" "$(openssl pkey -pubin < "$TEST_TMPDIR/backup.pub")"
}


@test "public key from private key" {
	pub=$($UBACK key pub <<eof
-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VuBCIEIOArTXPQaocIa+Y+WgvRc821Fr7Wzvsn3DR4mUAHrd5Q
-----END PRIVATE KEY-----
eof)

	expected_pub=$(cat <<eof
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VuAyEAeT+hiUrEev8AFB5IF8RU9XAPS7IK0iLwEMUJo6dqCAU=
-----END PUBLIC KEY-----
eof)

	assert_equal "$pub" "$expected_pub"
}
