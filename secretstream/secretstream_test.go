// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secretstream

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestLibsodiumCompatibility(t *testing.T) {
	var keyHex = "f964593f35bb282082d15daf2079b7d2884b80366ffd5b5426ae515818880b01"
	var headerHex = "598d48f7e699a43e3c456476b3124349b0559df34674f7ee"

	var expected = "9d85a512d07774ae808f7fd2e51f6b90096795dc90e5bd8e425f8744768318e2e25ff68a8d65b4555c09db416ef14483dfbeb11f2391bc1930420b866066cff9237895b648091f479c9f23cb4f33b0f6e623d1e3f947bea0cc074d890e9ba6d008e41425c9560ef18ce6816a874c2afce09d02ca50be7f11eb6c8522fde02a60dd76654b2b73dd767ecee52a93d59f6efe53144acb13ba393629531b46f5ba2c6cdb8f4b6fb223654f390214fec6543f32e9513e4144358e53bdf6e16b2d3922aa609161fb078b3c5a514bb1afa0a4f0cc7de10f620263070dd6337aa40917244965224a5849e8dfb588f25e6d5a7eaae68492ef6ddd9d42aadfaee46352c7a02f794e7699ee501ca9a58ffe93dcb9a85986b50fd53eae7e248434da645e54138a3a7c08136070f49fc32ce8dd9e62b1d239be53e110bb7f875f7d4944cbdf04ce24073eec5a1a4915367c77793a718a0e9ae540ce4a5f17fc4de4e9c6087ee40f298887b068696b7b18eb30270261400999ede4648a6cc052a426a854ed1f06a3493b760be4a02f817eee7a8124d0e0375ffce7762318231f4cca4bff524c419999cb903815dd82742dbfda8dc0e332e657ef07e2f566ebcf0f7c01eb51c90e18cc4121037de28db30ab1f4659a5f64f7017c8d075b4b8b86b0ff67d8a2d64bd16124cace1ff5c4ea8131221a8eabd347fc93e9c7a25ab5e637fdddff8ffefcf152bc8b7663d37325a4b7760609f896b7947abc408a643938b1faf4e2bc056962b3b9"

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		panic(err)
	}
	header, err := hex.DecodeString(headerHex)
	if err != nil {
		panic(err)
	}

	encryptor, err := NewEncryptor(header, key)
	if err != nil {
		t.Errorf("unable to initialize push state: %v.", err)
	}

	xof := sha3.NewShake256()
	xof.Write([]byte("secretstream test"))

	inputs := []struct {
		Size int
		Tag  byte
		// TODO does additional data actually change the output?
		AdditionalData []byte
	}{
		{10, TagMessage, []byte("additional data")},
		{64, TagMessage, nil},
		{1, TagMessage, nil},
		{127, TagRekey, nil},
		{33, TagMessage, nil},
		{128, TagRekey, []byte("more additional data")},
		{65, TagMessage, nil},
	}

	result := new(bytes.Buffer)
	for _, input := range inputs {
		data := make([]byte, input.Size)
		xof.Read(data)
		ctxt, err := encryptor.Push(data, input.AdditionalData, input.Tag)
		if err != nil {
			panic(err)
		}
		result.Write(ctxt)
	}

	actual := hex.EncodeToString(result.Bytes())
	if actual != expected {
		t.Fatalf("got %q, want %q", actual, expected)
	}
}

func checkedPush(t *testing.T, enc Encryptor, msg []byte, ad []byte, tag byte) []byte {
	ctxt, err := enc.Push(msg, ad, tag)
	if err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if len(ctxt) != len(msg)+additionalBytes {
		t.Fatalf("wrong ciphertext size: got %d, want %d", len(ctxt), len(msg)+additionalBytes)
	}
	return ctxt
}

func checkedPull(t *testing.T, dec Decryptor, ctxt []byte, ad []byte, expectedMsg []byte, expectedTag byte) {
	msg, tag, err := dec.Pull(ctxt, ad)
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}
	if tag != expectedTag {
		t.Fatalf("unexpected tag: got %x, want %x", tag, expectedTag)
	}
	if !bytes.Equal(msg, expectedMsg) {
		t.Fatalf("decrypted message does not match expected message")
	}
}

// Based on https://github.com/jedisct1/libsodium/blob/master/test/default/secretstream_xchacha20poly1305.c
func TestSecretStream(t *testing.T) {
	key := randomBytes(KeyBytes)
	header := randomBytes(HeaderBytes)
	m1 := randomBytes(randomInt(1000))
	m2 := randomBytes(randomInt(1000))
	m3 := randomBytes(randomInt(1000))
	additionalData := randomBytes(randomInt(100))

	encryptor, err := NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	c1 := checkedPush(t, encryptor, m1, nil, TagMessage)
	c2 := checkedPush(t, encryptor, m2, nil, TagMessage)
	c3 := checkedPush(t, encryptor, m3, additionalData, TagFinal)

	decryptor, err := NewDecryptor(header, key)
	if err != nil {
		t.Fatalf("NewDecryptor failed: %v", err)
	}

	checkedPull(t, decryptor, c1, nil, m1, TagMessage)
	checkedPull(t, decryptor, c2, nil, m2, TagMessage)

	if len(additionalData) > 0 {
		_, _, err := decryptor.Pull(c3, nil)
		if err == nil {
			t.Fatalf("expecting decryption to fail with empty additionalData")
		}
	}

	checkedPull(t, decryptor, c3, additionalData, m3, TagFinal)

	// Try to pull again after Final Tag.
	_, _, err = decryptor.Pull(c3, additionalData)
	if err == nil {
		t.Fatalf("expecting pull to fail after final tag")
	}

	_, _, err = decryptor.Pull(c2, nil)
	if err == nil {
		t.Fatalf("expecting out-of-order pull to fail")
	}
}

func TestTruncatedCiphertexts(t *testing.T) {
	key := randomBytes(KeyBytes)
	header := randomBytes(HeaderBytes)
	m1 := randomBytes(randomInt(1000))

	encryptor, err := NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}
	c1 := checkedPush(t, encryptor, m1, nil, TagMessage)

	decryptor, err := NewDecryptor(header, key)
	if err != nil {
		t.Fatalf("NewDecryptor failed: %v", err)
	}

	_, _, err = decryptor.Pull(c1[:16], nil)
	if err == nil {
		t.Fatal("expecting decryption failure for truncated ciphertext")
	}
	_, _, err = decryptor.Pull(c1[:additionalBytes], nil)
	if err == nil {
		t.Fatal("expecting decryption failure for truncated ciphertext")
	}
	_, _, err = decryptor.Pull(make([]byte, 0), nil)
	if err == nil {
		t.Fatal("expecting decryption failure for empty ciphertext")
	}

	checkedPull(t, decryptor, c1, nil, m1, TagMessage)
}

func TestDeterministic(t *testing.T) {
	key := randomBytes(KeyBytes)
	header := randomBytes(HeaderBytes)
	m1 := randomBytes(randomInt(1000))

	encryptor, err := NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	c1 := checkedPush(t, encryptor, m1, nil, TagMessage)
	c2 := checkedPush(t, encryptor, m1, nil, TagMessage)
	if bytes.Equal(c1, c2) {
		t.Fatal("ciphertexts should not match")
	}

	encryptor, err = NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}
	c1_ := checkedPush(t, encryptor, m1, nil, TagMessage)
	if !bytes.Equal(c1, c1_) {
		t.Fatal("push is not deterministic")
	}
}

func TestRekey(t *testing.T) {
	key := randomBytes(KeyBytes)
	header := randomBytes(HeaderBytes)
	m1 := randomBytes(randomInt(1000))
	m2 := randomBytes(randomInt(1000))

	encryptor, err := NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	// Push & pull two messages consecutively with explicit rekey
	c1 := checkedPush(t, encryptor, m1, nil, TagMessage)
	if err = encryptor.Rekey(); err != nil {
		t.Fatalf("rekey failed: %v", err)
	}
	c2 := checkedPush(t, encryptor, m2, nil, TagMessage)

	decryptor, err := NewDecryptor(header, key)
	if err != nil {
		t.Fatalf("NewDecryptor failed: %v", err)
	}
	checkedPull(t, decryptor, c1, nil, m1, TagMessage)

	_, _, err = decryptor.Pull(c2, nil)
	if err == nil {
		t.Fatalf("expecting decryption to fail without explicit rekey operation")
	}

	if err := decryptor.Rekey(); err != nil {
		t.Fatalf("rekey failed: %v", err)
	}
	checkedPull(t, decryptor, c2, nil, m2, TagMessage)

	// Push & Pull two consecutive messages with an explicit rekey using TagRekey
	encryptor, err = NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}
	c1 = checkedPush(t, encryptor, m1, nil, TagRekey)
	c2 = checkedPush(t, encryptor, m2, nil, TagMessage)

	decryptor, err = NewDecryptor(header, key)
	if err != nil {
		t.Fatalf("NewDecryptor failed: %v", err)
	}
	checkedPull(t, decryptor, c1, nil, m1, TagRekey)
	checkedPull(t, decryptor, c2, nil, m2, TagMessage)

	encryptor, err = NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}
	// Encrypt the first message again without forcing a rekey operation
	c1 = checkedPush(t, encryptor, m1, nil, TagMessage)
	// Encrypt the second message again without rekeyed state.
	c2_ := checkedPush(t, encryptor, m2, nil, TagMessage)
	// The two cipher texts for the second message must be different.
	if bytes.Equal(c2, c2_) {
		t.Fatal("expecting ciphertexts to be different")
	}

	encryptor, err = NewEncryptor(header, key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}
	c1 = checkedPush(t, encryptor, m1, nil, TagPush)

	// Force a counter overflow, check that the key has been updated
	// even though the tag was not changed to REKEY
	underlying := encryptor.(*streamState)
	underlying.counter = 0xffffffff

	stateCopy := *underlying

	c2 = checkedPush(t, encryptor, m2, nil, TagMessage)
	if bytes.Equal(stateCopy.key[:], underlying.key[:]) {
		t.Fatalf("keys match when they should not")
	}
	if stateCopy.counter == underlying.counter {
		t.Fatalf("counters match when they should not")
	}
	if underlying.counter != 1 {
		t.Fatalf("incorrect nonce")
	}

	decryptor, err = NewDecryptor(header, key)
	if err != nil {
		t.Fatalf("NewDecryptor failed: %v", err)
	}
	checkedPull(t, decryptor, c1, nil, m1, TagPush)

	underlying = decryptor.(*streamState)
	underlying.counter = 0xffffffff

	checkedPull(t, decryptor, c2, nil, m2, TagMessage)
}

func randomBytes(size int) []byte {
	x := make([]byte, size)
	if _, err := rand.Read(x); err != nil {
		panic(err)
	}
	return x
}

func randomInt(max int) int {
	b, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(b.Int64())
}
