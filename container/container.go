package container

import (
	"github.com/sloonz/uback/lib"

	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
	"github.com/klauspost/compress/zstd"
)

var (
	magic                 = "github.com/sloonz/uback/v0\n"
	ErrInvalidMagicHeader = errors.New("invalid magic header")
	ErrInvalidHeaderHash  = errors.New("invalid header hash")
)

// Encode into uback format
type Writer struct {
	w  io.Writer
	aw io.WriteCloser
	zw *zstd.Encoder
}

func NewWriter(w io.Writer, pk age.Recipient, typ string, compressionLevel int) (*Writer, error) {
	hdr := bytes.NewBuffer(nil)
	hdr.WriteString(magic)
	hdr.WriteString(fmt.Sprintf("type=%s,compression=zstd\n", typ))
	_, err := w.Write(hdr.Bytes())
	if err != nil {
		return nil, err
	}

	aw, err := age.Encrypt(w, pk)
	if err != nil {
		return nil, err
	}

	hdrHash := sha256.Sum256(hdr.Bytes())
	_, err = aw.Write(hdrHash[:])
	if err != nil {
		return nil, err
	}

	zw, err := zstd.NewWriter(aw, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)))
	if err != nil {
		return nil, err
	}

	return &Writer{
		w:  w,
		aw: aw,
		zw: zw,
	}, nil
}

// Part of io.WriteCloser interface
func (w *Writer) Write(p []byte) (int, error) {
	return w.zw.Write(p)
}

// Part of io.WriteCloser interface
// Note that this will write remaining buffered data to the underlying writer.
func (w *Writer) Close() error {
	err := w.zw.Close()
	if err != nil {
		return err
	}

	return w.aw.Close()
}

// Decoder for uback format
type Reader struct {
	r       io.Reader
	br      *bufio.Reader
	ar      io.Reader
	zr      *zstd.Decoder
	hdrHash [sha256.Size]byte
	Options *uback.Options
}

func NewReader(r io.Reader) (*Reader, error) {
	m := make([]byte, len(magic))
	_, err := io.ReadFull(r, m)
	if err != nil {
		return nil, err
	}
	if string(m) != magic {
		return nil, ErrInvalidMagicHeader
	}

	br := bufio.NewReader(r)
	optionsLine, err := br.ReadString('\n')
	if err != nil {
		return nil, err
	}
	opts, err := uback.EvalOptions(uback.SplitOptions(strings.TrimSpace(optionsLine)), make(map[string][]uback.KeyValuePair))
	if err != nil {
		return nil, err
	}

	hdr := bytes.NewBufferString(magic)
	hdr.WriteString(optionsLine)
	hdrHash := sha256.Sum256(hdr.Bytes())

	return &Reader{
		r:       r,
		br:      br,
		hdrHash: hdrHash,
		Options: opts,
	}, nil
}

// Prepares the decryption process. This must be called before any Read() call
func (r *Reader) Unseal(sk age.Identity) error {
	var err error
	r.ar, err = age.Decrypt(r.br, sk)
	if err != nil {
		return err
	}

	var encryptedHash [sha256.Size]byte
	_, err = io.ReadFull(r.ar, encryptedHash[:])
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare(encryptedHash[:], r.hdrHash[:]) == 0 {
		return ErrInvalidHeaderHash
	}

	r.zr, err = zstd.NewReader(r.ar)
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
