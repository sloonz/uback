// Original source: https://go-review.googlesource.com/c/crypto/+/288969/

// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package secretstream encrypts a sequence of messages, or a single message
split into an arbitrary number of chunks, using a secret key.

Secretstream uses ChaCha20 and Poly1305 to encrypt and authenticate messages
with secret-key cryptography. The length of messages is not hidden.

This API can be used to securely send an ordered sequence of messages to a
peer.  Since the length of the stream is not limited, it can also be used to
encrypt files regardless of their size.

It transparently generates nonces and automatically handles key rotation.

Secretstream provides the following properties:

1. Messages cannot be truncated, removed, reordered, duplicated or modified
without this being detected by the decryption functions.

2. The same sequence encrypted twice will produce different ciphertexts.

3. An authentication tag is added to each encrypted message: stream corruption
will be detected early, without having to read the stream until the end.

4. Each message can include additional data (ex: timestamp, protocol version)
in the computation of the authentication tag.

5. Messages can have different sizes.

6. There are no practical limits to the total length of the stream, or to the
total number of individual messages.

7. Ratcheting: at any point in the stream, it is possible to "forget" the key
used to encrypt the previous messages, and switch to a new key.
*/
package secretstream

import (
	"encoding/binary"
	"errors"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/poly1305"
)

const (
	// KeyBytes is the length of the secretstream key.
	KeyBytes = chacha20poly1305.KeySize

	HeaderBytes = 24
	// AdditionalBytes is the length of additional data bytes
	// plus 1 byte for the tag.
	additionalBytes = 17
	inputBytes      = 16
)

const (
	// TagMessage is the most common tag and doesn't add any information
	// about the nature of the message.
	TagMessage = 0x00
	// TagPush indicates that the message marks the end of a set of
	// messages but not the end of the stream.
	TagPush = 0x01
	// TagRekey indicates to "forget" the key used to encrypt this message
	// and the previous ones, and derive a new secret key.
	TagRekey = 0x02
	// TagFinal indicates that the message marks the end of the stream,
	// and erases the secret key used to encrypt the previous sequence.
	TagFinal = TagPush | TagRekey
)

var zeros [16]byte

type streamState struct {
	key     [KeyBytes]byte
	inonce  [8]byte
	counter uint32
}

type Encryptor interface {
	// Push encrypts a message and returns the ciphertext. Additional data is
	// optional and used for authentication; the tag specifies the nature of
	// the message in the stream.
	Push(message, additionalData []byte, tag byte) ([]byte, error)

	// Rekey performs explicit rekeying (i.e., it generates a new key used in
	// subsequent calls to Push), but doesn't add any information about the key
	// change to the stream. If this function is used to create an encrypted
	// stream, the decryption process must call that function at the exact same
	// location in the stream.
	Rekey() error
}

type Decryptor interface {
	// Pull decrypts a ciphertext, verifies the (optional) additional data,
	// and returns the plaintext and corresponding tag. Pull returns an error
	// if the ciphertext or additional data can not be authenticated.
	Pull(ctxt, additionalData []byte) ([]byte, byte, error)

	// Rekey performs explicit rekeying. See Encryptor.Rekey.
	Rekey() error
}

// NewEncryptor creates an encryption stream. The header must be HeaderBytes
// long must never be reused with the same key (like a nonce). The header must
// be sent/stored before the sequence of encrypted messages, as it is required
// to decrypt the stream. The header content doesn't have to be secret and
// decryption with a different header would fail.
func NewEncryptor(header, key []byte) (Encryptor, error) {
	return newState(header, key)
}

// NewDecryptor creates a decryption stream. The key and header must match
// what was used to encrypt the stream.
func NewDecryptor(header, key []byte) (Decryptor, error) {
	return newState(header, key)
}

func newState(header, key []byte) (*streamState, error) {
	if len(key) != KeyBytes {
		return nil, errors.New("secretstream: wrong key size")
	}
	if len(header) != HeaderBytes {
		return nil, errors.New("secretstream: wrong header size")
	}

	derived, err := chacha20.HChaCha20(key, header[:16]) // use only header's first 16 bytes as nonce
	if err != nil {
		return nil, err
	}

	ss := new(streamState)
	copy(ss.key[:], derived)
	copy(ss.inonce[:], header[inputBytes:])

	ss.resetCounter()

	return ss, nil
}

func (ss *streamState) Push(message, additionalData []byte, tag byte) ([]byte, error) {
	mlen := uint64(len(message))
	ctxt := make([]byte, mlen+additionalBytes)

	cipher, err := chacha20.NewUnauthenticatedCipher(ss.key[:], ss.nonce())
	if err != nil {
		return nil, err
	}

	var block [64]byte
	cipher.XORKeyStream(block[:], block[:])

	var polyBlock [32]byte
	copy(polyBlock[:], block[:32])
	oneTimeAuth := poly1305.New(&polyBlock)

	adlen := uint64(len(additionalData))
	oneTimeAuth.Write(additionalData)

	// Add padding to ensure 16-byte block length for more efficient block-aligned implementation:
	// https://tools.ietf.org/html/draft-irtf-cfrg-chacha20-poly1305-08#section-2.8
	// https://mailarchive.ietf.org/arch/msg/cfrg/u734TEOSDDWyQgE0pmhxjdncwvw/
	numZerosToWrite := (0x10 - adlen) & 0xf
	oneTimeAuth.Write(zeros[:numZerosToWrite])

	clear(block[:])
	block[0] = tag
	cipher.XORKeyStream(block[:], block[:])
	oneTimeAuth.Write(block[:])
	ctxt[0] = block[0]

	c := ctxt[1:]
	cipher.XORKeyStream(c, message)
	oneTimeAuth.Write(c[:mlen])

	numZerosToWrite = (0x10 + mlen - uint64(len(block))) & 0xf
	oneTimeAuth.Write(zeros[:numZerosToWrite])

	err = binary.Write(oneTimeAuth, binary.LittleEndian, adlen)
	if err != nil {
		return nil, err
	}

	err = binary.Write(oneTimeAuth, binary.LittleEndian, uint64(len(block))+mlen)
	if err != nil {
		return nil, err
	}

	mac := c[mlen:mlen]
	mac = oneTimeAuth.Sum(mac)

	xorBytes(ss.inonce[:], mac)
	ss.counter++

	if tag&TagRekey != 0 || ss.counter == 0 {
		ss.Rekey()
	}

	return ctxt, nil
}

func (ss *streamState) Pull(ctxt, additionalData []byte) ([]byte, byte, error) {
	clen := uint64(len(ctxt))
	if clen < additionalBytes {
		return nil, 0, errors.New("insufficiently long ciphertext")
	}

	cipher, err := chacha20.NewUnauthenticatedCipher(ss.key[:], ss.nonce())
	if err != nil {
		return nil, 0, err
	}

	var block [64]byte
	cipher.XORKeyStream(block[:], block[:])

	var polyBlock [32]byte
	copy(polyBlock[:], block[:32])
	oneTimeAuth := poly1305.New(&polyBlock)

	adlen := uint64(len(additionalData))
	oneTimeAuth.Write(additionalData)

	numZerosToWrite := (0x10 - adlen) & 0xf
	oneTimeAuth.Write(zeros[:numZerosToWrite])

	clear(block[:])
	block[0] = ctxt[0]
	cipher.XORKeyStream(block[:], block[:])

	tag := block[0]
	block[0] = ctxt[0]
	oneTimeAuth.Write(block[:])

	mlen := clen - additionalBytes
	c := ctxt[1:]
	oneTimeAuth.Write(c[:mlen])

	numZerosToWrite = (0x10 + mlen - uint64(len(block))) & 0xf
	oneTimeAuth.Write(zeros[:numZerosToWrite])

	err = binary.Write(oneTimeAuth, binary.LittleEndian, adlen)
	if err != nil {
		return nil, 0, err
	}
	err = binary.Write(oneTimeAuth, binary.LittleEndian, uint64(len(block))+mlen)
	if err != nil {
		return nil, 0, err
	}

	storedMac := c[mlen:]
	if !oneTimeAuth.Verify(storedMac) {
		return nil, 0, errors.New("incorrect MAC")
	}

	message := make([]byte, mlen)
	cipher.XORKeyStream(message, c[:mlen])

	xorBytes(ss.inonce[:], storedMac[:])
	ss.counter++

	if tag&TagRekey != 0 || ss.counter == 0 {
		ss.Rekey()
	}

	return message, tag, nil
}

func (ss *streamState) Rekey() error {
	newKeyAndINonce := make([]byte, KeyBytes+len(ss.inonce))

	copy(newKeyAndINonce[:KeyBytes], ss.key[:])
	copy(newKeyAndINonce[KeyBytes:], ss.inonce[:])

	cipher, err := chacha20.NewUnauthenticatedCipher(ss.key[:], ss.nonce())
	if err != nil {
		return err
	}

	cipher.XORKeyStream(newKeyAndINonce[:], newKeyAndINonce[:])

	copy(ss.key[:], newKeyAndINonce[:KeyBytes])
	copy(ss.inonce[:], newKeyAndINonce[KeyBytes:])

	ss.resetCounter()

	return nil
}

func (ss *streamState) nonce() []byte {
	nonce := make([]byte, 12)
	binary.LittleEndian.PutUint32(nonce[0:4], ss.counter)
	copy(nonce[4:], ss.inonce[:])
	return nonce
}

func (ss *streamState) resetCounter() {
	ss.counter = 1
}

func clear(data []byte) {
	for i := range data {
		data[i] = 0x00
	}
}

func xorBytes(output, input []byte) {
	for i := range output {
		output[i] ^= input[i]
	}
}
