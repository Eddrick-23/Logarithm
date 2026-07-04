package attack

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand/v2"
	"sync"
	"sync/atomic"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/generator"
	"github.com/klauspost/compress/zstd"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

// holds memory pools and handles encoding
type PayloadFactory struct {
	randPool    sync.Pool
	gzipPool    sync.Pool
	bufferPool  sync.Pool
	zstdEncoder *zstd.Encoder
	pool        []generator.LogRecordTemplate
	cfg         *config.CleanConfig
}

func NewPayloadFactory(cfg *config.CleanConfig) (*PayloadFactory, error) {
	var counter atomic.Uint64

	fmt.Printf("setting up payload factory. Generating randomised pool of size: %v \n", cfg.PoolSize)
	customRand := rand.New(rand.NewPCG(uint64(cfg.Seed), 1))
	logPool, err := generator.GenerateLogRecordPool(customRand, cfg)

	if err != nil {
		return nil, fmt.Errorf("failed to generate pool: %w", err)
	}
	fmt.Println("randomised pool generated")

	var zEncoder *zstd.Encoder
	if cfg.Encoding == "zstd" {
		var err error
		zEncoder, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd encoder: %v", err)
		}
	}

	return &PayloadFactory{
		pool:        logPool,
		cfg:         cfg,
		zstdEncoder: zEncoder,
		randPool: sync.Pool{
			New: func() any {
				n := counter.Add(1)
				return rand.New(rand.NewPCG(uint64(cfg.Seed)+n, 1))
			},
		},
		gzipPool: sync.Pool{
			New: func() any {
				if !(cfg.Encoding == "gzip") {
					return nil
				}

				return gzip.NewWriter(io.Discard)
			},
		},
		bufferPool: sync.Pool{
			New: func() any {
				if !(cfg.Encoding == "gzip") {
					return nil
				}
				return new(bytes.Buffer)
			},
		},
	}, nil
}

func (p *PayloadFactory) GenerateEncodedPayload() ([]byte, error) {
	raw, err := p.generatePayload()
	if err != nil {
		return nil, err
	}

	encoded, err := p.encodePayload(raw)

	if err != nil {
		return nil, err
	}

	return encoded, nil
}

// helper to gzip compress and set tgt body and header
func (p *PayloadFactory) encodeGzip(payload []byte) ([]byte, error) {
	buf := p.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer p.bufferPool.Put(buf)

	gz := p.gzipPool.Get().(*gzip.Writer)
	gz.Reset(buf)
	defer p.gzipPool.Put(gz)

	if _, err := gz.Write(payload); err != nil {
		return nil, fmt.Errorf("gzip write error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close error: %w", err)
	}

	return bytes.Clone(buf.Bytes()), nil
}

func (p *PayloadFactory) encodeZstd(payload []byte) []byte {
	buf := make([]byte, 0, len(payload))
	compressed := p.zstdEncoder.EncodeAll(payload, buf)

	return compressed
}

func (p *PayloadFactory) generatePayload() ([]byte, error) {
	r := p.randPool.Get().(*rand.Rand)
	defer p.randPool.Put(r)
	req := plogotlp.NewExportRequestFromLogs(generator.GenerateRequest(r, p.cfg, p.pool))
	switch p.cfg.ContentType {
	case "proto":
		return req.MarshalProto()
	case "json":
		return req.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported content type %s", p.cfg.ContentType)
	}
}

func (p *PayloadFactory) encodePayload(payload []byte) ([]byte, error) {
	switch p.cfg.Encoding {
	case "zstd":
		return p.encodeZstd(payload), nil
	case "gzip":
		return p.encodeGzip(payload)
	default:
		return payload, nil
	}
}
