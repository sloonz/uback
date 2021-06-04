load "helpers/bats-support/load"
load "helpers/bats-assert/load"

UBACK="$BATS_TEST_DIRNAME/../uback"
TEST_TMPDIR="$BATS_RUN_TMPDIR/key"

setup() {
	mkdir -p "$TEST_TMPDIR"
}

@test "key generation" {
	$UBACK key gen "$TEST_TMPDIR/backup.key" "$TEST_TMPDIR/backup.pub"
	assert_equal "$(cat "$TEST_TMPDIR/backup.pub")" "$($UBACK key pub < "$TEST_TMPDIR/backup.key")"
}


@test "public key from private key" {
	pub=$($UBACK key pub <<< "AGE-SECRET-KEY-1FZM50PS7W57CZV4EZVFVZZHVPK02Q6WNC0FU3DZ9RHLLYQY42PZQNDKJZW")
	expected_pub="age1fu6nhq9cvjezr6lffnnfj3txqvxdsv0est5vqzamujcfnj80jfpqdcj87k"
	assert_equal "$pub" "$expected_pub"
}
