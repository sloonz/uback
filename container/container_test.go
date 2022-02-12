package container

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestReadWriter(t *testing.T) {
	sk, err := age.GenerateX25519Identity()
	if err != nil {
		t.Error(err)
		return
	}
	pk := sk.Recipient()
	m := "In cryptography, Curve25519 is an elliptic curve offering 128 bits of security (256 bits key size) and designed for use with the elliptic curve Diffie-Hellman (ECDH) key agreement scheme." +
		"It is one of the fastest ECC curves and is not covered by any known patents."

	buf := bytes.NewBuffer(nil)
	w, err := NewWriter(buf, pk, "test", 3)
	if err != nil {
		t.Errorf("cannot create writer: %v", err)
		return
	}

	for i := 0; i < 100; i++ {
		n, err := w.Write([]byte(m))
		if err != nil || n != len(m) {
			t.Errorf("cannot write plaintext: %v", err)
			return
		}
	}

	err = w.Close()
	if err != nil {
		t.Errorf("cannot close encryptor: %v", err)
		return
	}

	r, err := NewReader(buf)
	if err != nil {
		t.Errorf("cannot create reader: %v", err)
		return
	}

	if r.Options.String["Type"] != "test" {
		t.Errorf("type mismatch; expected: test, got: %v", r.Options.String["Type"])
		return
	}

	err = r.Unseal(sk)
	if err != nil {
		t.Errorf("cannot unseal reader: %v", err)
	}

	m2, err := io.ReadAll(r)
	if err != nil {
		t.Errorf("cannot decrypt: %v", err)
		return
	}

	err = r.Close()
	if err != nil {
		t.Error(err)
	}

	if string(m2) != strings.Repeat(m, 100) {
		t.Errorf("different plaintext; expected: %v, got: %v", m, m2)
	}
}

// The following backup file was generated randomly
//   dd if=/dev/urandom bs=65536 count=5 > raw
// (sha-256 hash of raw: 0b24b8319e35171b713a66fa3cf4922f542883436f93e0959c655dca2b01e96e)
// and created with this script:
// (printf "github.com/sloonz/uback/v0\ntype=test,compression=zstd\n"; (node -e 'process.stdout.write(crypto.createHash("sha256").update("github.com/sloonz/uback/v0\ntype=test,compression=zstd\n").digest())'; zstd -1 < raw) | age -r age1qn6y7pg9cr5vd92xy46rd9dkjfjumr4s0v8xesn3w95zurq2mu7s4pezh7) > test.ubkp

//go:embed test.ubkp
var testBackup []byte

// Test that we can correctly recover the raw hash
func TestAgeZstd(t *testing.T) {
	r, err := NewReader(bytes.NewBuffer(testBackup))
	if err != nil {
		t.Error(err)
		return
	}

	identities, err := age.ParseIdentities(bytes.NewBufferString("AGE-SECRET-KEY-1WLX59L52P29SJPUD8XE4NVMCZ2CD3KXYR62890PFTVTVYJRL29UST5LG03"))
	if err != nil {
		t.Error(err)
		return
	}

	err = r.Unseal(identities[0])
	if err != nil {
		t.Error(err)
		return
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
		return
	}

	hashBytes := sha256.Sum256(data)
	hash := hex.EncodeToString(hashBytes[:])
	expectedHash := "0b24b8319e35171b713a66fa3cf4922f542883436f93e0959c655dca2b01e96e"
	if hash != expectedHash {
		t.Errorf("bad output; expected hash: %v, got: %v", expectedHash, hash)
	}
}

// Test that tampering with header is detected
func TestTamperedHeader(t *testing.T) {
	tamperedTestBackup := make([]byte, len(testBackup))
	copy(tamperedTestBackup, testBackup)
	tamperedTestBackup[len("github.com/sloonz/uback/v0\ntype=")] = 'W'
	r, err := NewReader(bytes.NewBuffer(tamperedTestBackup))
	if err != nil {
		t.Error(err)
		return
	}

	identities, err := age.ParseIdentities(bytes.NewBufferString("AGE-SECRET-KEY-1WLX59L52P29SJPUD8XE4NVMCZ2CD3KXYR62890PFTVTVYJRL29UST5LG03"))
	if err != nil {
		t.Error(err)
		return
	}

	err = r.Unseal(identities[0])
	if !errors.Is(err, ErrInvalidHeaderHash) {
		t.Errorf("expected ErrInvalidHeaderHash, got %v", err)
		return
	}
}

// Test that tampering with payload is detected
func TestTamperedPayload(t *testing.T) {
	tamperedTestBackup := make([]byte, len(testBackup))
	copy(tamperedTestBackup, testBackup)
	tamperedTestBackup[65535*2] ^= 0xff
	r, err := NewReader(bytes.NewBuffer(tamperedTestBackup))
	if err != nil {
		t.Error(err)
		return
	}

	identities, err := age.ParseIdentities(bytes.NewBufferString("AGE-SECRET-KEY-1WLX59L52P29SJPUD8XE4NVMCZ2CD3KXYR62890PFTVTVYJRL29UST5LG03"))
	if err != nil {
		t.Error(err)
		return
	}

	err = r.Unseal(identities[0])
	if err != nil {
		t.Error(err)
		return
	}

	_, err = io.ReadAll(r)
	if err == nil {
		t.Error("expected decryption error, got none")
	}
}

func TestUnencrypted(t *testing.T) {
	m := "In cryptography, Curve25519 is an elliptic curve offering 128 bits of security (256 bits key size) and designed for use with the elliptic curve Diffie-Hellman (ECDH) key agreement scheme." +
		"It is one of the fastest ECC curves and is not covered by any known patents."

	buf := bytes.NewBuffer(nil)
	w, err := NewWriter(buf, nil, "test", 3)
	if err != nil {
		t.Errorf("cannot create writer: %v", err)
		return
	}

	for i := 0; i < 100; i++ {
		n, err := w.Write([]byte(m))
		if err != nil || n != len(m) {
			t.Errorf("cannot write plaintext: %v", err)
			return
		}
	}

	err = w.Close()
	if err != nil {
		t.Errorf("cannot close container: %v", err)
		return
	}

	r, err := NewReader(buf)
	if err != nil {
		t.Errorf("cannot create reader: %v", err)
		return
	}

	if r.Options.String["Type"] != "test" {
		t.Errorf("type mismatch; expected: test, got: %v", r.Options.String["Type"])
		return
	}

	err = r.Unseal(nil)
	if err != nil {
		t.Errorf("cannot unseal reader: %v", err)
	}

	m2, err := io.ReadAll(r)
	if err != nil {
		t.Errorf("cannot read: %v", err)
		return
	}

	err = r.Close()
	if err != nil {
		t.Error(err)
	}

	if string(m2) != strings.Repeat(m, 100) {
		t.Errorf("different plaintext; expected: %v, got: %v", m, m2)
	}
}
