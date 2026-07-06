package worker

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/storage"
	"github.com/Eddrick-23/Logarithm/internal/transport"
)

const WorkerName = "worker"

type DecompressorFactory func() (Decompressor, error)
type job func(Decompressor)

type WorkerPool struct {
	logger        *slog.Logger
	store         storage.LogStore
	decoder       Decoder
	transformer   Transformer
	publisher     Publisher
	estimatedRows int
	numWorkers    int
	jobs          chan job
}

type Config struct {
	Logger        *slog.Logger
	Store         storage.LogStore
	DecompFactory DecompressorFactory
	Decoder       Decoder
	Transformer   Transformer
	Publisher     Publisher
	EstimatedRows int
	NumWorkers    int
}

const (
	defaultEstimatedRows = 1000
	defaultNumWorkers    = 1
)

func NewWorkerPool(cfg Config) (*WorkerPool, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	if cfg.EstimatedRows <= 0 {
		cfg.Logger.Warn("Estimated rows must be positive, defaulting to 1000")
		cfg.EstimatedRows = defaultEstimatedRows
	}

	if cfg.NumWorkers <= 0 {
		cfg.Logger.Warn("NumWorkers must be positive, defaulting to 1")
		cfg.NumWorkers = defaultNumWorkers
	}

	jobs := make(chan job, cfg.NumWorkers)

	// worker will take in functions from the chan and call them
	// error handling is managed within the function.
	// Fanning in i.e. combining results, is handled within the function
	// via appending to the same appender
	for i := range cfg.NumWorkers {
		logger := cfg.Logger.With("component", "PoolWorker"+strconv.Itoa(i+1))
		decompressor, err := cfg.DecompFactory()
		if err != nil {
			return nil, err
		}
		go func() {
			for job := range jobs {
				runSafely(logger, func() { job(decompressor) })
			}
		}()
	}

	return &WorkerPool{
		cfg.Logger,
		cfg.Store,
		cfg.Decoder,
		cfg.Transformer,
		cfg.Publisher,
		cfg.EstimatedRows,
		cfg.NumWorkers,
		jobs,
	}, nil
}

func (w *WorkerPool) Close() {
	// closes job channel so worker goroutines can exit
	close(w.jobs)
}

func (w *WorkerPool) ConsumeCallback() (func([]transport.Message) error, error) {
	return func(messages []transport.Message) error {
		if len(messages) == 0 {
			return nil
		}

		appender := w.store.FastInsert(w.estimatedRows)
		intervals := splitMessages(len(messages), w.numWorkers)
		var wg sync.WaitGroup
		wg.Add(len(intervals))

		for _, iv := range intervals {
			w.jobs <- func(decompressor Decompressor) {
				defer wg.Done()
				processMessages(
					w.logger,
					decompressor,
					w.decoder,
					messages[iv.start:iv.end],
					w.transformer,
					w.publisher,
					appender,
				)
			}
		}
		wg.Wait()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		return appender.Flush(ctx)
	}, nil
}

// allows recovery of goroutine worker if it panics while running the job
func runSafely(logger *slog.Logger, job func()) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic in worker job", "panic", r)
		}
	}()

	job()
}

// represents start(inclusive), end(exclusive): [start, end)
type interval struct {
	start, end int
}

// splitMessages divides numMessagse into at most numWorkers contiguous,
// non-overlapping regions. Sizes differ by at most one message, with
// remainder distributed across the different regions. Returns index
// bounds wrapped in an interval struct.
func splitMessages(numMessages, numWorkers int) []interval {
	numWorkers = min(numMessages, numWorkers)

	if numWorkers <= 0 {
		return nil
	}

	intervals := make([]interval, numWorkers)
	base := numMessages / numWorkers
	remainder := numMessages % numWorkers

	start := 0
	for i := 0; i < numWorkers; i++ {
		size := base
		if i < remainder {
			size++
		}
		intervals[i] = interval{start: start, end: start + size}
		start += size
	}

	return intervals
}

func processMessages(logger *slog.Logger, decompressor Decompressor, decoder Decoder, messages []transport.Message,
	transformer Transformer, publisher Publisher, appender storage.LogAppender) {

	for _, msg := range messages {
		decompressedPayload, cleanup, err := decompressor.decompress(msg.Payload, msg.Headers)
		if err != nil {
			logger.Error("dropped log payload due to decompress error", "err", err)
			continue
		}

		req, err := decoder.decode(decompressedPayload, msg.Headers)
		if err != nil {
			logger.Error("dropped malformed log payload", "err", err)
			cleanup()
			continue
		}
		cleanup()

		logs := req.Logs()
		for i := 0; i < logs.ResourceLogs().Len(); i++ {
			transformer.Flatten(logs.ResourceLogs().At(i), publisher, appender)
		}
	}
}

func DLQCallback(producer transport.Producer, subject string) func([]byte, map[string][]string) error {

	return func(payload []byte, headers map[string][]string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return producer.PublishLogs(ctx, subject, payload, headers)
	}
}

func DelayCalculator(backoff []time.Duration) func(uint64) time.Duration {
	return func(deliveredCount uint64) time.Duration {
		if len(backoff) == 0 {
			return 0
		}
		if int(deliveredCount) >= len(backoff) {
			return backoff[len(backoff)-1]
		}
		idx := max(int(deliveredCount)-1, 0)
		return backoff[idx]
	}
}
