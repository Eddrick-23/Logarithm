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

type gzipResources struct {
	gzReader  *gzip.Reader
	bufReader *bytes.Reader
	buf       *bytes.Buffer
}

type LogDecompressor struct {
	bufPool     sync.Pool
	zstdDecoder *zstd.Decoder
	gzipPool    sync.Pool
}

type decompressorConfig struct {
	zstdOpts []zstd.DOption
}

type DecompressorOption func(*decompressorConfig)

// Set zstdConcurrencyLimit to bound number of possible workers when running decodes.
// This is useful when having multiple workers to reduce context switching.
// Setting to 0 uses zstd defualts: min(4, GOMAXPROCS) as per zstd docs.
func WithZstdConcurrencyLimit(limit int) DecompressorOption {
	return func(dc *decompressorConfig) {
		if limit > 0 {
			dc.zstdOpts = append(dc.zstdOpts, zstd.WithDecoderConcurrency(limit))
		}
	}
}

func NewLogDecompressor(opts ...DecompressorOption) (*LogDecompressor, error) {
	cfg := &decompressorConfig{}

	for _, opt := range opts {
		opt(cfg)
	}

	zstdDecoder, err := zstd.NewReader(nil, cfg.zstdOpts...)
	if err != nil {
		return nil, err
	}

	return &LogDecompressor{
		zstdDecoder: zstdDecoder,
		bufPool: sync.Pool{
			New: func() any {
				// prealloc 64kb
				b := make([]byte, 0, 64*1024)
				return &b
			},
		},
		gzipPool: sync.Pool{
			New: func() any {
				return &gzipResources{
					gzReader:  new(gzip.Reader),
					bufReader: new(bytes.Reader),
					buf:       new(bytes.Buffer),
				}
			},
		},
	}, nil
}

// decompress decompresses payload according to the Content-Encoding header.
//
// Returns the decompressed bytes, a cleanup callback, and an error. The
// returned slice is only valid until cleanup is called. Callers must
// finish reading it (e.g. fully unmarshal it) before calling cleanup, since
// gzip-encoded payloads alias a pooled buffer that may be reused once
// released. Cleanup is always safe to call exactly once; on error it is a
// no-op and may be omitted.
func (d *LogDecompressor) decompress(payload []byte, headers map[string][]string) ([]byte, func(), error) {
	encoding := ""
	if vals, ok := headers["Content-Encoding"]; ok && len(vals) > 0 {
		encoding = vals[0]
	}

	if encoding == "" || encoding == "none" || encoding == "identity" {
		return payload, func() {}, nil
	}

	switch encoding {
	case "zstd":
		return d.decompressZstd(payload)
	case "gzip":
		return d.decompressGzip(payload)
	default:
		return payload, func() {}, fmt.Errorf("unsupported encoding")
	}
}

// decompressZstd decompresses a zstd-encoded payload using a pooled buffer.
// The returned slice aliases the pooled buffer and is only valid until
// the returned cleanup callback is called.
func (d *LogDecompressor) decompressZstd(payload []byte) ([]byte, func(), error) {
	bufPtr := d.bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0] // clear byte slice

	buf, err := d.zstdDecoder.DecodeAll(payload, buf)

	if err != nil {
		d.bufPool.Put(bufPtr)
		return nil, func() {}, err
	}

	*bufPtr = buf // in case underlying slice was reallocated
	callback := func() {
		d.bufPool.Put(bufPtr)
	}

	return buf, callback, err
}

// decompressGzip decompresses payload using pooled gzip resources.
// The returned slice aliases the pooled buffer and is only valid until
// the returned cleanup callback is called.
func (d *LogDecompressor) decompressGzip(payload []byte) ([]byte, func(), error) {
	gzResources := d.gzipPool.Get().(*gzipResources)
	gzResources.bufReader.Reset(payload)

	if err := gzResources.gzReader.Reset(gzResources.bufReader); err != nil {
		d.gzipPool.Put(gzResources)
		return nil, func() {}, fmt.Errorf("gzip reset error: %w", err)
	}

	gzResources.buf.Reset()
	if _, err := io.Copy(gzResources.buf, gzResources.gzReader); err != nil {
		gzResources.gzReader.Close() // best effort, copy error takes priority
		d.gzipPool.Put(gzResources)
		return nil, func() {}, fmt.Errorf("gzip copy error: %w", err)
	}

	// handle close explicitly in case issues transferring to dst
	if err := gzResources.gzReader.Close(); err != nil {
		d.gzipPool.Put(gzResources)
		return nil, func() {}, fmt.Errorf("gzip close error: %w", err)
	}

	callback := func() {
		d.gzipPool.Put(gzResources)
	}

	return gzResources.buf.Bytes(), callback, nil
}
