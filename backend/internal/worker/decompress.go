package worker

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type Decompressor interface {
	decompress(payload []byte, headers map[string][]string) ([]byte, func(), error)
}

var _ Decompressor = (*LogDecompressor)(nil)

type LogDecompressor struct {
	zstdDecoder    *zstd.Decoder
	gzipReaderPool sync.Pool
	bufPool        sync.Pool
}

func NewLogDecompressor() (*LogDecompressor, error) {
	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	return &LogDecompressor{
		zstdDecoder: zstdDecoder,
		gzipReaderPool: sync.Pool{
			New: func() any {
				return new(gzip.Reader)
			},
		},
		bufPool: sync.Pool{
			New: func() any {
				// prealloc 64kb
				b := make([]byte, 0, 64*1024)
				return &b
			},
		},
	}, nil
}

// decompress payload using zstd or gzip
//
// uses internal pooled buffers for efficient reuse
// returns decompressed bytes, cleanup callback, error
// cleanup callback returns the byte slice back to the buffer
func (d *LogDecompressor) decompress(payload []byte, headers map[string][]string) ([]byte, func(), error) {
	encoding := ""
	if vals, ok := headers["Content-Encoding"]; ok && len(vals) > 0 {
		encoding = vals[0]
	}

	if encoding == "" || encoding == "none" {
		return payload, func() {}, nil
	}

	bufPtr := d.bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0] // clear byte slice

	var err error

	switch encoding {
	case "zstd":
		buf, err = d.zstdDecoder.DecodeAll(payload, buf)
	case "gzip":
		buf, err = d.decompressGzip(payload, buf)
	default:
		d.bufPool.Put(bufPtr)
		return payload, func() {}, nil
	}

	if err != nil {
		d.bufPool.Put(bufPtr)
		return nil, func() {}, err
	}

	cleanup := func() {
		*bufPtr = buf // update pointer incase underlying buffer grew
		d.bufPool.Put(bufPtr)

	}
	return buf, cleanup, nil
}

// decompressGzip decompresses payload into dst, reusing its backing
// array where possible. It returns the resulting slice
func (d *LogDecompressor) decompressGzip(payload, dst []byte) ([]byte, error) {
	gz := d.gzipReaderPool.Get().(*gzip.Reader)
	defer d.gzipReaderPool.Put(gz)

	payloadReader := bytes.NewReader(payload)
	if err := gz.Reset(payloadReader); err != nil {
		return nil, fmt.Errorf("gzip reset error: %w", err)
	}

	buf := bytes.NewBuffer(dst)
	if _, err := io.Copy(buf, gz); err != nil {
		gz.Close() // best effort, copy error takes priority
		return nil, fmt.Errorf("gzip copy error: %w", err)
	}

	// handle close explicitly in case issues transferring to dst
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close error: %w", err)
	}

	return buf.Bytes(), nil
}
