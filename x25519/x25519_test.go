package x25519

import (
	"bytes"
	"encoding/pem"
	"strings"
	"testing"
)

var (
	skPEM = strings.TrimSpace(`
-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VuBCIEIFjhWA2+cwrC7ClOVKszbpEDf0TiDryHTLrZKEcJlXty
-----END PRIVATE KEY-----`)

	pkPEM = strings.TrimSpace(`
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VuAyEAdZY0n430RDh0zYzAJr/MvWFoxa0Rjg0L67lSaxDueGE=
-----END PUBLIC KEY-----`)

	skBytes     = []byte{0x58, 0xe1, 0x58, 0x0d, 0xbe, 0x73, 0x0a, 0xc2, 0xec, 0x29, 0x4e, 0x54, 0xab, 0x33, 0x6e, 0x91, 0x03, 0x7f, 0x44, 0xe2, 0x0e, 0xbc, 0x87, 0x4c, 0xba, 0xd9, 0x28, 0x47, 0x09, 0x95, 0x7b, 0x72}
	pkBytes     = []byte{0x75, 0x96, 0x34, 0x9f, 0x8d, 0xf4, 0x44, 0x38, 0x74, 0xcd, 0x8c, 0xc0, 0x26, 0xbf, 0xcc, 0xbd, 0x61, 0x68, 0xc5, 0xad, 0x11, 0x8e, 0x0d, 0x0b, 0xeb, 0xb9, 0x52, 0x6b, 0x10, 0xee, 0x78, 0x61}
	peerSkBytes = []byte{0x78, 0xa9, 0xaa, 0x24, 0x45, 0x5d, 0xaa, 0x2c, 0x3c, 0x96, 0xfb, 0xf9, 0x5a, 0x14, 0x2c, 0x87, 0xee, 0x54, 0x79, 0xee, 0x63, 0xf7, 0x97, 0x4a, 0x7a, 0x3f, 0xcc, 0xf3, 0xa6, 0x88, 0xfe, 0x5b}
	peerPkBytes = []byte{0x6b, 0x32, 0x69, 0x0d, 0x6b, 0xe0, 0x3d, 0x13, 0x6c, 0x4e, 0x25, 0x8b, 0xb2, 0xb5, 0xf0, 0x01, 0x36, 0x67, 0x18, 0x6a, 0xee, 0x07, 0x87, 0x7d, 0x78, 0x65, 0x92, 0x38, 0x9d, 0xd6, 0x69, 0x73}
	dhResult    = []byte{0x49, 0x7e, 0x0b, 0x22, 0x19, 0x59, 0xa6, 0xe6, 0x3d, 0x18, 0x37, 0xa8, 0xf0, 0x01, 0x76, 0xf8, 0xf8, 0xc1, 0xa7, 0xe7, 0xfc, 0xe0, 0xfe, 0x01, 0x2c, 0x79, 0x6a, 0x50, 0x71, 0x3b, 0x0a, 0x1d}
)

func TestMarshal(t *testing.T) {
	skDER, err := PrivateKey(skBytes).Marshal()
	if err != nil {
		t.Errorf("failed to marshal private key: %v", err)
	} else {
		skPEMRes := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: skDER,
		})))

		if skPEMRes != skPEM {
			t.Errorf("Failed to encode PEM: expected: %v, result: %v", skPEM, skPEMRes)
		}
	}

	pkDER, err := PublicKey(pkBytes).Marshal()
	if err != nil {
		t.Errorf("failed to marshal public key: %v", err)
	} else {
		pkPEMRes := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pkDER,
		})))

		if pkPEMRes != pkPEM {
			t.Errorf("Failed to encode PEM: expected: %v, result: %v", pkPEM, pkPEMRes)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	skDER, _ := pem.Decode([]byte(skPEM))
	sk, err := ParsePrivateKey(skDER.Bytes)
	if err != nil {
		t.Errorf("Failed to decode private key: %v", err)
	} else {
		if !bytes.Equal([]byte(sk), skBytes) {
			t.Errorf("Failed to decode private key: expected: %v, got: %v", skBytes, []byte(sk))
		}
	}

	pkDER, _ := pem.Decode([]byte(pkPEM))
	pk, err := ParsePublicKey(pkDER.Bytes)
	if err != nil {
		t.Errorf("Failed to decode public key: %v", err)
	} else {
		if !bytes.Equal([]byte(pk), pkBytes) {
			t.Errorf("Failed to decode public key: expected: %v, got: %v", pkBytes, []byte(pk))
		}
	}
}

func TestPublic(t *testing.T) {
	pk, err := PrivateKey(skBytes).Public()
	if err != nil {
		t.Errorf("failed to derive public key from private key: %v", err)
	} else {
		if !bytes.Equal(pkBytes, []byte(pk)) {
			t.Errorf("failed to derive public key from private key: expected: %v, got: %v", pkBytes, []byte(pk))
		}
	}
}

func TestDH(t *testing.T) {
	dh, err := PrivateKey(skBytes).GenerateSessionKey(peerPkBytes)
	if err != nil {
		t.Errorf("DH error: %v", err)
	} else {
		if !bytes.Equal(dh, dhResult) {
			t.Errorf("faild to compute DH: expected: %v, got: %v", dhResult, dh)
		}
	}

	dh, err = PrivateKey(peerSkBytes).GenerateSessionKey(pkBytes)
	if err != nil {
		t.Errorf("DH error: %v", err)
	} else {
		if !bytes.Equal(dh, dhResult) {
			t.Errorf("faild to compute DH: expected: %v, got: %v", dhResult, dh)
		}
	}
}

func TestLoadKeysStandardEncoding(t *testing.T) {
	// Standard encoding
	sk, err := LoadPrivateKey("", "MC4CAQAwBQYDK2VuBCIEIFjhWA2+cwrC7ClOVKszbpEDf0TiDryHTLrZKEcJlXty")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(sk, skBytes) {
		t.Errorf("cannot load private key; expected: %v, got: %v", skBytes, sk)
	}

	pk, err := LoadPublicKey("", "MCowBQYDK2VuAyEAdZY0n430RDh0zYzAJr/MvWFoxa0Rjg0L67lSaxDueGE=")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}

	pk, err = LoadPublicKey("", "MCowBQYDK2VuAyEAdZY0n430RDh0zYzAJr/MvWFoxa0Rjg0L67lSaxDueGE")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}

	pk, err = LoadPublicKey("", "MC4CAQAwBQYDK2VuBCIEIFjhWA2+cwrC7ClOVKszbpEDf0TiDryHTLrZKEcJlXty")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}

	// Standard encoding, no padding
	pk, err = LoadPublicKey("", "MCowBQYDK2VuAyEAdZY0n430RDh0zYzAJr/MvWFoxa0Rjg0L67lSaxDueGE")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}
}

func TestLoadKeysURLEncoding(t *testing.T) {
	// URL encoding
	sk, err := LoadPrivateKey("", "MC4CAQAwBQYDK2VuBCIEIFjhWA2-cwrC7ClOVKszbpEDf0TiDryHTLrZKEcJlXty")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(sk, skBytes) {
		t.Errorf("cannot load private key; expected: %v, got: %v", skBytes, sk)
	}

	pk, err := LoadPublicKey("", "MCowBQYDK2VuAyEAdZY0n430RDh0zYzAJr_MvWFoxa0Rjg0L67lSaxDueGE=")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}

	pk, err = LoadPublicKey("", "MCowBQYDK2VuAyEAdZY0n430RDh0zYzAJr_MvWFoxa0Rjg0L67lSaxDueGE")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}

	pk, err = LoadPublicKey("", "MC4CAQAwBQYDK2VuBCIEIFjhWA2-cwrC7ClOVKszbpEDf0TiDryHTLrZKEcJlXty")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}

	// URL encoding, no padding
	pk, err = LoadPublicKey("", "MCowBQYDK2VuAyEAdZY0n430RDh0zYzAJr_MvWFoxa0Rjg0L67lSaxDueGE")
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(pk, pkBytes) {
		t.Errorf("cannot load public key; expected: %v, got: %v", pkBytes, pk)
	}
}
