# uback File Format, Version 1

A `uback` backup (default extension: `.ubkp`) contains the following fields, in that order :

* The magic string `UBKP1` (4 bytes)
* `Flags`, little-endian unsigned integer (2 bytes)
* `TypeLength`, little-endian unsigned integer (4 bytes)
* `Type`, string (TypeLength bytes)
* `Header`, 24 bytes
* `PublicKey`, 32 bytes
* `EphemeralPublicKey`, 32 bytes

If `Header`, `PublicKey` and `EphemeralPublicKey` are all all-zero
bytes, then rest of the file is unencrypted. Otherwise,
the rest of the file consists of a [libsodium
secretstream](https://libsodium.gitbook.io/doc/secret-key_cryptography/secretstream)
of 4096-bytes messages (plaintext length, 4113 ciphertext blocks). The
`Header` corresponds to the `header` parameter of the stream. For the
stream key, see the "Key Derivation" section.

`Flags` indicates the compression of the payload. `0` means no
compression, `1` means zstd.

Unencrypted and uncompressed backups are planned but not yet implemented.

## Key Derivation

The key derivation process is the same as the one used by libsodium in
its `crypto_box_seal` API.

Namely, the owner of the private key associated with the public key in
the `PublicKey` field can derive the `secretstream` key by applying the
following procedure :

* Extract the `EphemeralPublicKey` field as a X25519 public key (now
called `epk`)

* Load the private key associated with the public key described in the
`PublicKey` field (now called `sk`)

* Compute the shared secret by applying X25519 DH with `epk` and `sk`
(now called `s`)

* The `secretstream` key is the result of the `HChaCha20` function with
`s` as the key and an all-zero input as the nonce.
