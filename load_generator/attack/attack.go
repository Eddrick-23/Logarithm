package attack

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/generator"
	"github.com/klauspost/compress/zstd"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

type Attack struct {
	randPool    sync.Pool
	gzipPool    sync.Pool
	bufferPool  sync.Pool
	zstdEncoder *zstd.Encoder
	pool        []generator.LogRecordTemplate
	cfg         *config.CleanConfig
	attacker    *vegeta.Attacker
	duration    time.Duration
}

func NewAttack(cfg *config.CleanConfig, duration time.Duration) (*Attack, error) {
	var counter atomic.Uint64

	fmt.Printf("setting up attack. Generating randomised pool of size: %v \n", cfg.PoolSize)
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

	return &Attack{
		attacker:    vegeta.NewAttacker(),
		pool:        logPool,
		cfg:         cfg,
		duration:    duration,
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

// helper to gzip compress and set tgt body and header
func (a *Attack) encodeGzip(payload []byte) ([]byte, error) {
	buf := a.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer a.bufferPool.Put(buf)

	gz := a.gzipPool.Get().(*gzip.Writer)
	gz.Reset(buf)
	defer a.gzipPool.Put(gz)

	if _, err := gz.Write(payload); err != nil {
		return nil, fmt.Errorf("gzip write error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close error: %w", err)
	}

	return bytes.Clone(buf.Bytes()), nil
}

func (a *Attack) encodeZstd(payload []byte) []byte {
	buf := make([]byte, 0, len(payload))
	compressed := a.zstdEncoder.EncodeAll(payload, buf)

	return compressed
}

func (a *Attack) generatePayload() ([]byte, error) {
	r := a.randPool.Get().(*rand.Rand)
	defer a.randPool.Put(r)
	req := plogotlp.NewExportRequestFromLogs(generator.GenerateRequest(r, a.cfg, a.pool))
	switch a.cfg.ContentType {
	case "proto":
		return req.MarshalProto()
	case "json":
		return req.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported content type %s", a.cfg.ContentType)
	}
}

func (a *Attack) encodePayload(payload []byte) ([]byte, error) {
	switch a.cfg.Encoding {
	case "zstd":
		return a.encodeZstd(payload), nil
	case "gzip":
		return a.encodeGzip(payload)
	default:
		return payload, nil
	}
}

func (a *Attack) writeHeaders(tgt *vegeta.Target) {
	if tgt.Header == nil {
		tgt.Header = make(http.Header)
	}

	switch a.cfg.ContentType {
	case "proto":
		tgt.Header.Set("Content-Type", "application/x-protobuf")
	case "json":
		tgt.Header.Set("Content-Type", "application/json")
	default:
		// don't set
	}

	switch a.cfg.Encoding {
	case "zstd":
		tgt.Header.Set("Content-Encoding", "zstd")
	case "gzip":
		tgt.Header.Set("Content-Encoding", "gzip")
	default:
		tgt.Header.Set("Content-Encoding", "identity")
	}
}

func (a *Attack) Start() <-chan *vegeta.Result {
	rate := vegeta.Rate{Freq: a.cfg.Rps, Per: time.Second}

	targeter := func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		payload, err := a.generatePayload()

		if err != nil {
			return fmt.Errorf("error marshaling payload: %w", err)
		}

		encoded, err := a.encodePayload(payload)

		if err != nil {
			return fmt.Errorf("error encoding payload: %w", err)
		}

		a.writeHeaders(tgt)

		tgt.Body = encoded
		tgt.Method = a.cfg.Method
		tgt.URL = a.cfg.TargetUrl
		return nil
	}
	return a.attacker.Attack(targeter, rate, a.duration, "logarithm load generator")
}

func (a *Attack) Stop() {
	a.attacker.Stop()
}
