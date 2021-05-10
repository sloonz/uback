package container

import (
	"github.com/sloonz/uback/secretstream"
	"github.com/sloonz/uback/x25519"

	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/crypto/chacha20"
)

const (
	defaultBlockSize     = 4096
	flagsCompressionZstd = 1
	magic                = "UBK1"
)

var zero = make([]byte, 16)

func memzero(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}

func deriveSharedKey(pk x25519.PublicKey, sk x25519.PrivateKey) ([]byte, error) {
	s, err := sk.GenerateSessionKey(pk)
	if err != nil {
		return nil, err
	}

	return chacha20.HChaCha20(s, zero)
}

// Encode into uback format
type Writer struct {
	w           io.Writer
	sw          *secretstreamWriter
	zw          *zstd.Encoder
	header      []byte
	wroteHeader bool
}

func NewWriter(w io.Writer, pk *x25519.PublicKey, typ string, compressionLevel int) (*Writer, error) {
	return newWriter(w, pk, typ, compressionLevel, defaultBlockSize)
}

func newWriter(w io.Writer, pk *x25519.PublicKey, typ string, compressionLevel, blockSize int) (*Writer, error) {
	ssHeader := make([]byte, secretstream.HeaderBytes)
	_, err := io.ReadFull(rand.Reader, ssHeader)
	if err != nil {
		return nil, err
	}

	epk, esk, err := x25519.GenerateKey()
	if err != nil {
		return nil, err
	}

	key, err := deriveSharedKey(*pk, esk)
	if err != nil {
		return nil, err
	}
	memzero(esk)

	sw, err := secretstream.NewEncryptor(ssHeader, key)
	if err != nil {
		return nil, err
	}
	memzero(key)

	headerBuf := bytes.NewBufferString(magic)
	binary.Write(headerBuf, binary.LittleEndian, uint16(flagsCompressionZstd))
	binary.Write(headerBuf, binary.LittleEndian, uint32(len(typ)))
	headerBuf.WriteString(typ)
	swWriter := newSecretstreamWriter(w, sw, blockSize, headerBuf.Bytes())
	headerBuf.Write(ssHeader)
	headerBuf.Write(*pk)
	headerBuf.Write(epk)

	zw, err := zstd.NewWriter(swWriter, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)))
	if err != nil {
		return nil, err
	}

	return &Writer{
		w:      w,
		zw:     zw,
		sw:     swWriter,
		header: headerBuf.Bytes(),
	}, nil
}

// Part of io.WriteCloser interface
func (w *Writer) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
		_, err := w.w.Write(w.header)
		if err != nil {
			return 0, err
		}
	}

	return w.zw.Write(p)
}

// Part of io.WriteCloser interface
// Note that this will write remaining buffered data to the underlying writer.
func (w *Writer) Close() error {
	err := w.zw.Close()
	if err != nil {
		return err
	}

	return w.sw.Close()
}

// Header of a backup file
type Header struct {
	Type      string           // Type of the source (for example tar)
	PublicKey x25519.PublicKey // Public key associated to the private key intended to decrypt the backup
	header    []byte
	epk       x25519.PublicKey
	flags     uint16
}

// Decoder for uback format
type Reader struct {
	Header
	r         io.Reader
	sr        *secretstreamReader
	zr        *zstd.Decoder
	blockSize int
}

func NewReader(r io.Reader) (*Reader, error) {
	return newReader(r, defaultBlockSize)
}

func newReader(r io.Reader, blockSize int) (*Reader, error) {
	m := make([]byte, 4)
	_, err := io.ReadFull(r, m)
	if err != nil {
		return nil, err
	}
	if string(m) != magic {
		return nil, fmt.Errorf("invalid magic header")
	}

	var flags uint16
	err = binary.Read(r, binary.LittleEndian, &flags)
	if err != nil {
		return nil, err
	}
	if flags != flagsCompressionZstd {
		return nil, errors.New("unsupported flags")
	}

	var typeLen uint32
	err = binary.Read(r, binary.LittleEndian, &typeLen)
	if err != nil {
		return nil, err
	}

	typeBuf := make([]byte, typeLen)
	_, err = io.ReadFull(r, typeBuf)
	if err != nil {
		return nil, err
	}

	ssHeader := make([]byte, secretstream.HeaderBytes)
	_, err = io.ReadFull(r, ssHeader)
	if err != nil {
		return nil, err
	}

	pk := make([]byte, x25519.PublicKeyBytes)
	_, err = io.ReadFull(r, pk)
	if err != nil {
		return nil, err
	}

	epk := make([]byte, x25519.PublicKeyBytes)
	_, err = io.ReadFull(r, epk)
	if err != nil {
		return nil, err
	}

	return &Reader{
		r:         r,
		blockSize: blockSize,
		Header: Header{
			Type:      string(typeBuf),
			PublicKey: pk,
			epk:       epk,
			header:    ssHeader,
			flags:     flags,
		},
	}, nil
}

// Prepares the decryption process. This must be called before any Read() call
func (r *Reader) Unseal(sk x25519.PrivateKey) error {
	pk, err := sk.Public()
	if err != nil {
		return err
	}

	if !bytes.Equal(pk, r.Header.PublicKey) {
		return fmt.Errorf("provided private key does not correspond to public key used for encryption")
	}

	key, err := deriveSharedKey(r.Header.epk, sk)
	if err != nil {
		return err
	}

	dec, err := secretstream.NewDecryptor(r.Header.header, key)
	if err != nil {
		return err
	}
	memzero(key)

	adBuf := bytes.NewBufferString(magic)
	binary.Write(adBuf, binary.LittleEndian, r.Header.flags)
	binary.Write(adBuf, binary.LittleEndian, uint32(len(r.Header.Type)))
	adBuf.WriteString(r.Header.Type)

	r.sr = newSecretstreamReader(r.r, dec, r.blockSize, adBuf.Bytes())
	r.zr, err = zstd.NewReader(r.sr)
	if err != nil {
		return err
	}

	return nil
}

// Part of io.ReadCloser interface
func (r *Reader) Read(p []byte) (int, error) {
	return r.zr.Read(p)
}

// Part of io.ReadCloser interface
func (r *Reader) Close() error {
	if r.zr != nil {
		r.zr.Close()
	}
	return nil
}
