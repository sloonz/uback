package container

import (
	"github.com/sloonz/uback/secretstream"
	"github.com/sloonz/uback/x25519"

	"bytes"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"io"
	"strings"
	"testing"
)

func TestDeriveSharedKey(t *testing.T) {
	skBytes := []byte{0x58, 0xe1, 0x58, 0x0d, 0xbe, 0x73, 0x0a, 0xc2, 0xec, 0x29, 0x4e, 0x54, 0xab, 0x33, 0x6e, 0x91, 0x03, 0x7f, 0x44, 0xe2, 0x0e, 0xbc, 0x87, 0x4c, 0xba, 0xd9, 0x28, 0x47, 0x09, 0x95, 0x7b, 0x72}
	peerPkBytes := []byte{0x6b, 0x32, 0x69, 0x0d, 0x6b, 0xe0, 0x3d, 0x13, 0x6c, 0x4e, 0x25, 0x8b, 0xb2, 0xb5, 0xf0, 0x01, 0x36, 0x67, 0x18, 0x6a, 0xee, 0x07, 0x87, 0x7d, 0x78, 0x65, 0x92, 0x38, 0x9d, 0xd6, 0x69, 0x73}
	expectedSharedBytes := []byte{0x85, 0x5c, 0xf, 0xed, 0x30, 0xfc, 0x3e, 0x91, 0x70, 0x9b, 0x82, 0x8a, 0x20, 0xd1, 0xbc, 0x8d, 0xaa, 0x94, 0x42, 0x8, 0xbf, 0x73, 0xeb, 0xe8, 0x78, 0x86, 0xbd, 0x87, 0x3f, 0x28, 0xca, 0xe6}

	sharedBytes, err := deriveSharedKey(peerPkBytes, skBytes)
	if err != nil {
		t.Errorf("cannot derive shared key: %v", err)
	}

	if !bytes.Equal(sharedBytes, expectedSharedBytes) {
		t.Errorf("cannot derive shared key: expected: %v, got: %v", expectedSharedBytes, sharedBytes)
	}
}

func TestReadWriter(t *testing.T) {
	sk := x25519.PrivateKey([]byte{0x58, 0xe1, 0x58, 0x0d, 0xbe, 0x73, 0x0a, 0xc2, 0xec, 0x29, 0x4e, 0x54, 0xab, 0x33, 0x6e, 0x91, 0x03, 0x7f, 0x44, 0xe2, 0x0e, 0xbc, 0x87, 0x4c, 0xba, 0xd9, 0x28, 0x47, 0x09, 0x95, 0x7b, 0x72})
	pk := x25519.PublicKey([]byte{0x75, 0x96, 0x34, 0x9f, 0x8d, 0xf4, 0x44, 0x38, 0x74, 0xcd, 0x8c, 0xc0, 0x26, 0xbf, 0xcc, 0xbd, 0x61, 0x68, 0xc5, 0xad, 0x11, 0x8e, 0x0d, 0x0b, 0xeb, 0xb9, 0x52, 0x6b, 0x10, 0xee, 0x78, 0x61})
	m := "In cryptography, Curve25519 is an elliptic curve offering 128 bits of security (256 bits key size) and designed for use with the elliptic curve Diffie-Hellman (ECDH) key agreement scheme." +
		"It is one of the fastest ECC curves and is not covered by any known patents."

	buf := bytes.NewBuffer(nil)
	w, err := newWriter(buf, &pk, "test", 3, 64)
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

	r, err := newReader(buf, 64)
	if err != nil {
		t.Errorf("cannot create reader: %v", err)
		return
	}

	if r.Header.Type != "test" {
		t.Errorf("type mismatch; expected: test, got: %v", r.Header.Type)
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

func TestBlockBoundary(t *testing.T) {
	key := []byte{0x85, 0x5c, 0xf, 0xed, 0x30, 0xfc, 0x3e, 0x91, 0x70, 0x9b, 0x82, 0x8a, 0x20, 0xd1, 0xbc, 0x8d, 0xaa, 0x94, 0x42, 0x8, 0xbf, 0x73, 0xeb, 0xe8, 0x78, 0x86, 0xbd, 0x87, 0x3f, 0x28, 0xca, 0xe6}

	ssHeader := make([]byte, secretstream.HeaderBytes)
	_, err := io.ReadFull(rand.Reader, ssHeader)
	if err != nil {
		t.Error(err)
		return
	}

	sw, err := secretstream.NewEncryptor(ssHeader, key)
	if err != nil {
		t.Error(err)
		return
	}

	buf := bytes.NewBuffer(nil)
	w := newSecretstreamWriter(buf, sw, 64, nil)
	_, err = w.Write([]byte(strings.Repeat("a", 64) + strings.Repeat("b", 64)))
	if err != nil {
		t.Error(err)
		return
	}
	_, err = w.Write([]byte(strings.Repeat("c", 64) + strings.Repeat("d", 64)))
	if err != nil {
		t.Error(err)
		return
	}
	err = w.Close()
	if err != nil {
		t.Error(err)
		return
	}

	dec, err := secretstream.NewDecryptor(ssHeader, key)
	if err != nil {
		t.Error(err)
		return
	}
	r := newSecretstreamReader(buf, dec, 64, nil)
	m, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
		return
	}

	expected := strings.Repeat("a", 64) + strings.Repeat("b", 64) + strings.Repeat("c", 64) + strings.Repeat("d", 64)
	if string(m) != expected {
		t.Errorf("different plaintext; expected: %v, got: %v", expected, m)
	}
}

// The following backup file was generated randomly
//   dd if=/dev/urandom bs=1024 count=5 > raw
// (sha-256 hash of raw: 0529e36fcd1410ecfd79da63521f85249f43384f00880ea342a1c0b2db0dc260)
// and then compressed with zstd
//   zstd -1 < raw > compressed
// and encrypted with this C program:
//
// #include <stdio.h>
// #include <sodium.h>
//
// #define CHUNK_SIZE 4096
//
// unsigned char key[crypto_secretstream_xchacha20poly1305_KEYBYTES] = {0x85, 0x5c, 0xf, 0xed, 0x30, 0xfc, 0x3e, 0x91, 0x70, 0x9b, 0x82, 0x8a, 0x20, 0xd1, 0xbc, 0x8d, 0xaa, 0x94, 0x42, 0x8, 0xbf, 0x73, 0xeb, 0xe8, 0x78, 0x86, 0xbd, 0x87, 0x3f, 0x28, 0xca, 0xe6};
// unsigned char prolog[] = {0x55, 0x42, 0x4b, 0x31, 0x01, 0x00, 0x04, 0x00, 0x00, 0x00, 0x74, 0x65, 0x73, 0x74};
// unsigned char pk[] = {0x6b, 0x32, 0x69, 0x0d, 0x6b, 0xe0, 0x3d, 0x13, 0x6c, 0x4e, 0x25, 0x8b, 0xb2, 0xb5, 0xf0, 0x01, 0x36, 0x67, 0x18, 0x6a, 0xee, 0x07, 0x87, 0x7d, 0x78, 0x65, 0x92, 0x38, 0x9d, 0xd6, 0x69, 0x73};
// unsigned char epk[] = {0x75, 0x96, 0x34, 0x9f, 0x8d, 0xf4, 0x44, 0x38, 0x74, 0xcd, 0x8c, 0xc0, 0x26, 0xbf, 0xcc, 0xbd, 0x61, 0x68, 0xc5, 0xad, 0x11, 0x8e, 0x0d, 0x0b, 0xeb, 0xb9, 0x52, 0x6b, 0x10, 0xee, 0x78, 0x61};
//
// int main(void) {
//    if (sodium_init() != 0) {
//        return 1;
//    }
//
//    unsigned char  buf_in[CHUNK_SIZE];
//    unsigned char  buf_out[CHUNK_SIZE + crypto_secretstream_xchacha20poly1305_ABYTES];
//    unsigned char  header[crypto_secretstream_xchacha20poly1305_HEADERBYTES];
//    crypto_secretstream_xchacha20poly1305_state st;
//    unsigned long long out_len;
//    size_t         rlen;
//    int            eof;
//    unsigned char  tag;
//    int            block = 0;
//
//    crypto_secretstream_xchacha20poly1305_init_push(&st, header, key);
//    fwrite(prolog, 1, sizeof prolog, stdout);
//    fwrite(header, 1, sizeof header, stdout);
//    fwrite(pk, 1, sizeof pk, stdout);
//    fwrite(epk, 1, sizeof epk, stdout);
//    do {
//        rlen = fread(buf_in, 1, sizeof buf_in, stdin);
//        eof = feof(stdin);
//        tag = eof ? crypto_secretstream_xchacha20poly1305_TAG_FINAL : 0;
//        if (block == 0) {
//            crypto_secretstream_xchacha20poly1305_push(&st, buf_out, &out_len, buf_in, rlen,
//                                                       &prolog[0], 14, tag);
//        } else {
//            crypto_secretstream_xchacha20poly1305_push(&st, buf_out, &out_len, buf_in, rlen,
//                                                       NULL, 0, tag);
//        }
//        fwrite(buf_out, 1, (size_t) out_len, stdout);
//        block++;
//    } while (!eof);
//
//    fclose(stdout);
//    fclose(stdin);
//
//    return 0;
// }

//go:embed test.ubkp
var testBackup []byte

// Test that we can correctly recover the raw hash
func TestSodiumZstd(t *testing.T) {
	r, err := NewReader(bytes.NewBuffer(testBackup))
	if err != nil {
		t.Error(err)
		return
	}

	sk := x25519.PrivateKey([]byte{0x78, 0xa9, 0xaa, 0x24, 0x45, 0x5d, 0xaa, 0x2c, 0x3c, 0x96, 0xfb, 0xf9, 0x5a, 0x14, 0x2c, 0x87, 0xee, 0x54, 0x79, 0xee, 0x63, 0xf7, 0x97, 0x4a, 0x7a, 0x3f, 0xcc, 0xf3, 0xa6, 0x88, 0xfe, 0x5b})
	err = r.Unseal(sk)
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
	expectedHash := "0529e36fcd1410ecfd79da63521f85249f43384f00880ea342a1c0b2db0dc260"
	if hash != expectedHash {
		t.Errorf("bad output; expected hash: %v, got: %v", expectedHash, hash)
	}
}
