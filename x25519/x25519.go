package x25519

import (
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/curve25519"
)

const (
	PublicKeyBytes  = 32
	PrivateKeyBytes = 32
	SessionKeyBytes = 32
)

var oidSignatureX25519 = asn1.ObjectIdentifier{1, 3, 101, 110}

type publicKeyInfo struct {
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

type pkcs8 struct {
	Version    int
	Algo       pkix.AlgorithmIdentifier
	PrivateKey []byte
}

// X25519 Public Key
type PublicKey []byte

// X25519 Private Key
type PrivateKey []byte

// Generate a random keypair
func GenerateKey() (PublicKey, PrivateKey, error) {
	sk := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, sk)
	if err != nil {
		return nil, nil, err
	}

	sk[0] &= 248
	sk[31] &= 127
	sk[31] |= 64

	pk, err := PrivateKey(sk).Public()
	if err != nil {
		return nil, nil, err
	}

	return pk, sk, err
}

// Generate a public key associated to a private key
func (priv PrivateKey) Public() (PublicKey, error) {
	return curve25519.X25519(priv, curve25519.Basepoint)
}

// Do not directly use the resulting key, hash it first
func (priv PrivateKey) GenerateSessionKey(peer PublicKey) ([]byte, error) {
	return curve25519.X25519(priv, peer)
}

// DER encoding of the private key
func (priv PrivateKey) Marshal() ([]byte, error) {
	marshaledPriv, err := asn1.Marshal(priv)
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(pkcs8{
		Algo: pkix.AlgorithmIdentifier{
			Algorithm: oidSignatureX25519,
		},
		PrivateKey: marshaledPriv,
	})
}

// DER encoding of the public key
func (pub PublicKey) Marshal() ([]byte, error) {
	return asn1.Marshal(publicKeyInfo{
		Algorithm: pkix.AlgorithmIdentifier{
			Algorithm: oidSignatureX25519,
		},
		PublicKey: asn1.BitString{
			Bytes:     pub,
			BitLength: 8 * len(pub),
		},
	})
}

// Load a private key from its DER encoding
func ParsePrivateKey(der []byte) (PrivateKey, error) {
	var privateKey pkcs8
	_, err := asn1.Unmarshal(der, &privateKey)
	if err != nil {
		return nil, err
	}

	if !oidSignatureX25519.Equal(privateKey.Algo.Algorithm) {
		return nil, fmt.Errorf("invalid private key type: %v", privateKey.Algo.Algorithm)
	}

	var privBytes []byte
	_, err = asn1.Unmarshal(privateKey.PrivateKey, &privBytes)
	if err != nil {
		return nil, err
	}

	if len(privBytes) != 32 {
		return nil, fmt.Errorf("invalid private key")
	}

	return privBytes, nil
}

// Load a public key from its DER encoding
func ParsePublicKey(der []byte) (PublicKey, error) {
	var publicKey publicKeyInfo

	_, err := asn1.Unmarshal(der, &publicKey)
	if err != nil {
		return nil, err
	}

	if !oidSignatureX25519.Equal(publicKey.Algorithm.Algorithm) {
		return nil, fmt.Errorf("invalid private key type: %v", publicKey.Algorithm.Algorithm)
	}

	pubBytes := publicKey.PublicKey.RightAlign()
	if len(pubBytes) != 32 {
		return nil, fmt.Errorf("invalid public key")
	}

	return pubBytes, nil
}

// Load a private key either from a PEM file (if keyFile argument is provided), or from its base64-encoded DER encoding (key argument)
func LoadPrivateKey(keyFile, key string) (PrivateKey, error) {
	if keyFile != "" && key != "" {
		return nil, fmt.Errorf("must provide one of key file or key, not both")
	}

	if keyFile != "" {
		pemData, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}

		der, _ := pem.Decode(pemData)
		sk, err := ParsePrivateKey(der.Bytes)
		if err != nil {
			return nil, err
		}

		return sk, nil
	}

	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		der, err := enc.DecodeString(key)
		if err != nil {
			continue
		}

		return ParsePrivateKey(der)
	}

	return nil, fmt.Errorf("cannot decode base64 encoded key")
}

// Load a private key either from a PEM file (if keyFile argument is provided), or from its base64-encoded DER encoding (key argument)
// The PEM file/DER data can also represent a X25519 private key, in which case the public key is derived from the private key
func LoadPublicKey(keyFile, key string) (PublicKey, error) {
	if keyFile != "" && key != "" {
		return nil, fmt.Errorf("must provide one of key file or key, not both")
	}

	if keyFile != "" {
		pemData, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}

		der, _ := pem.Decode(pemData)
		pk, err := ParsePublicKey(der.Bytes)
		if err != nil {
			sk, err2 := ParsePrivateKey(der.Bytes)
			if err2 != nil {
				return nil, err
			}

			return sk.Public()
		}

		return pk, nil
	}

	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		der, err := enc.DecodeString(key)
		if err != nil {
			continue
		}

		pk, err := ParsePublicKey(der)
		if err != nil {
			sk, err2 := ParsePrivateKey(der)
			if err2 != nil {
				return nil, err
			}

			return sk.Public()
		}

		return pk, nil
	}

	return nil, fmt.Errorf("cannot decode base64 encoded key")
}
