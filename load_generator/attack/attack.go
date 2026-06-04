package attack

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Eddrick-23/Logarithm/api/schemas"
	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/generator"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

type Attack struct {
	randPool   sync.Pool
	gzipPool   sync.Pool
	bufferPool sync.Pool
	pool       []schemas.LogRecordDTO
	cfg        *config.CleanConfig
	attacker   *vegeta.Attacker
	duration   time.Duration
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

	return &Attack{
		attacker: vegeta.NewAttacker(),
		pool:     logPool,
		cfg:      cfg,
		duration: duration,
		randPool: sync.Pool{
			New: func() any {
				n := counter.Add(1)
				return rand.New(rand.NewPCG(uint64(cfg.Seed)+n, 1))
			},
		},
		gzipPool: sync.Pool{
			New: func() any {
				if !cfg.Gzip {
					return nil
				}

				return gzip.NewWriter(io.Discard)
			},
		},
		bufferPool: sync.Pool{
			New: func() any {
				if !cfg.Gzip {
					return nil
				}
				return new(bytes.Buffer)
			},
		},
	}, nil
}

func (a *Attack) Start() <-chan *vegeta.Result {
	rate := vegeta.Rate{Freq: a.cfg.Rps, Per: time.Second}

	targeter := func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}
		r := a.randPool.Get().(*rand.Rand)
		defer a.randPool.Put(r)
		payload, err := json.Marshal(generator.GenerateRequest(r, a.cfg, a.pool))

		if err != nil {
			return fmt.Errorf("error marshaling json: %w", err)
		}

		if a.cfg.Gzip {
			buf := a.bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			defer a.bufferPool.Put(buf)

			gz := a.gzipPool.Get().(*gzip.Writer)
			gz.Reset(buf)
			defer a.gzipPool.Put(gz)

			if _, err := gz.Write(payload); err != nil {
				return fmt.Errorf("gzip write error: %w", err)
			}
			if err := gz.Close(); err != nil {
				return fmt.Errorf("gzip close error: %w", err)
			}

			tgt.Body = bytes.Clone(buf.Bytes())
			tgt.Header = http.Header{
				"Content-Type":     []string{"application/json"},
				"Content-Encoding": []string{"gzip"},
			}
		} else {
			tgt.Body = payload
			tgt.Header = http.Header{
				"Content-Type": []string{"application/json"},
			}
		}

		tgt.Method = a.cfg.Method
		tgt.URL = a.cfg.TargetUrl
		return nil
	}
	return a.attacker.Attack(targeter, rate, a.duration, "logarithm load generator")
}

func (a *Attack) Stop() {
	a.attacker.Stop()
}
