package container

import (
	"github.com/sloonz/uback/secretstream"

	"io"
)

const additionalBytes = 17

type secretstreamWriter struct {
	w                   io.Writer
	enc                 secretstream.Encryptor
	additionalData      []byte
	wroteAdditionalData bool
	buf                 []byte
}

func newSecretstreamWriter(w io.Writer, enc secretstream.Encryptor, blockSize int, additionalData []byte) *secretstreamWriter {
	return &secretstreamWriter{
		w:              w,
		enc:            enc,
		additionalData: additionalData,
		buf:            make([]byte, 0, blockSize),
	}
}

func (w *secretstreamWriter) Write(data []byte) (int, error) {
	blockSize := cap(w.buf)
	nn := 0

	if len(w.buf) > 0 {
		n := copy(w.buf[len(w.buf):blockSize], data)
		nn += n
		data = data[n:]
		w.buf = w.buf[:len(w.buf)+n]
		if len(w.buf) == blockSize {
			err := w.writeBlock(w.buf, 0)
			if err != nil {
				return nn, err
			}
			w.buf = w.buf[0:0]
		}
	}

	for len(data) >= blockSize {
		err := w.writeBlock(data[:blockSize], 0)
		if err != nil {
			return nn, err
		}
		data = data[blockSize:]
		nn += blockSize
	}

	if len(data) > 0 {
		n := copy(w.buf[:blockSize], data)
		nn += n
		w.buf = w.buf[:n]
	}

	return nn, nil
}

func (w *secretstreamWriter) writeBlock(data []byte, tag byte) error {
	var ad []byte
	if !w.wroteAdditionalData {
		w.wroteAdditionalData = true
		ad = w.additionalData
	}

	encData, err := w.enc.Push(data, ad, tag)
	if err != nil {
		return err
	}

	_, err = w.w.Write(encData)
	if err != nil {
		return err
	}

	return nil
}

func (w *secretstreamWriter) Close() error {
	return w.writeBlock(w.buf, secretstream.TagFinal)
}

type secretstreamReader struct {
	r                 io.Reader
	buf               []byte
	cbuf              []byte
	dec               secretstream.Decryptor
	additionalData    []byte
	readAdditonalData bool
	rpos              int
	eof               bool
}

func newSecretstreamReader(r io.Reader, dec secretstream.Decryptor, blockSize int, additionalData []byte) *secretstreamReader {
	return &secretstreamReader{
		r:              r,
		dec:            dec,
		additionalData: additionalData,
		cbuf:           make([]byte, blockSize+additionalBytes),
	}
}

func (r *secretstreamReader) Read(data []byte) (int, error) {
	var tag byte
	var ad []byte

	if len(data) == 0 {
		return 0, nil
	}

	if r.eof && len(r.buf) == 0 {
		return 0, io.EOF
	}

	nn := 0

	for len(data) > 0 {
		if len(r.buf) == 0 {
			if r.eof {
				return nn, nil
			}

			n, err := io.ReadFull(r.r, r.cbuf)
			if err != nil && err != io.ErrUnexpectedEOF {
				if err == io.EOF {
					return nn, io.ErrUnexpectedEOF
				}
				return nn, err
			}

			ad = nil
			if !r.readAdditonalData {
				r.readAdditonalData = true
				ad = r.additionalData
			}

			r.buf, tag, err = r.dec.Pull(r.cbuf[:n], ad)
			if err != nil {
				return nn, err
			}

			if tag == secretstream.TagFinal {
				r.eof = true
			}
		}

		n := copy(data, r.buf[r.rpos:])
		nn += n
		data = data[n:]
		r.rpos += n
		if len(r.buf) == r.rpos {
			r.rpos = 0
			r.buf = nil
		}
	}

	return nn, nil
}
