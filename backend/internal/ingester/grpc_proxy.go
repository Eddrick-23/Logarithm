package ingester

import (
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// CodecName is the gRPC content-subtype used to select rawCodec on both the
// client and server. Callers must set it via grpc.CallContentSubtype(ingester.CodecName)
// when invoking RPCs against the server returned by NewGRPCServer.
const CodecName = "raw-bytes"
const zstdName = "zstd"
const gzipName = "gzip"

func init() {
	encoding.RegisterCodecV2(&rawCodec{})
	encoding.RegisterCompressor(&noopCompressor{zstdName})
	encoding.RegisterCompressor(&noopCompressor{gzipName})
}

// RawFrame is the gRPC message type used by the raw-bytes passthrough codec.
// Unlike a typical generated proto message, RawFrame is never marshaled by
// reflection. rawCodec copies the wire-format bytes into RawBytes directly,
// so the ingester never has to know the shape of the underlying payload.
type RawFrame struct {
	RawBytes []byte
}

// rawCodec copies raw wire bytes into a RawFrame for the ingest passthrough
// path. For any other registered proto service on the server (e.g. the
// gRPC health check), it falls back to standard proto marshal/unmarshal.
// See Marshal/Unmarshal for the branching logic.
type rawCodec struct{}

func (r *rawCodec) Name() string {
	return CodecName
}

// Marshal returns the wire format of v. RawFrame values are passed through
// unchanged (raw ingest path); any other proto.Message falls back to
// standard proto marshaling (e.g. health check responses).
func (r *rawCodec) Marshal(v any) (mem.BufferSlice, error) {
	// fast path, any unregistered payloads (e.g. OTel payloads)
	if out, ok := v.(*RawFrame); ok {
		return mem.BufferSlice{mem.SliceBuffer(out.RawBytes)}, nil
	}

	// registered proto service (e.g. health check)
	if pm, ok := v.(proto.Message); ok {
		b, err := proto.Marshal(pm)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal proto message: %w", err)
		}
		return mem.BufferSlice{mem.SliceBuffer(b)}, nil
	}

	return nil, fmt.Errorf("unsupported type %T", v)
}

// Unmarshal parses the wire format into v. For the raw ingest path, v is a
// *RawFrame and we keep the bytes as is rather than decoding - the ingester
// doesn't need to understand the payload shape. For any other registered
// proto service (e.g. health check), v is a proto.Message and we decode
// normally.
func (r *rawCodec) Unmarshal(data mem.BufferSlice, v any) error {
	// fast path, any unregistered payloads (e.g. OTel payloads)
	if out, ok := v.(*RawFrame); ok {
		srcBytes := data.Materialize()
		out.RawBytes = append(out.RawBytes[:0], srcBytes...)
		return nil
	}

	// registered proto service (e.g. health check)
	if pm, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data.Materialize(), pm)
	}

	return fmt.Errorf("unsupported type %T", v)
}

// since we don't define any protobuf messages and associated methods
// there is no "routing" for the gRPC server
// By default, gRPC returns an UNIMPLEMENTED status code.
// Instead we can implement an UnknownServiceHandler and hand the connection
// stream directly to this handler.
type proxyHandler struct {
	logger            *slog.Logger
	producer          transport.Producer
	natsSubjectPrefix string
}

func newProxyHandler(logger *slog.Logger, producer transport.Producer, prefix string) *proxyHandler {
	return &proxyHandler{
		logger:            logger,
		producer:          producer,
		natsSubjectPrefix: prefix,
	}
}

func (p *proxyHandler) NewStreamHandler(presize int64) func(any, grpc.ServerStream) error {
	rawFramePool := sync.Pool{
		New: func() any {
			return &RawFrame{
				RawBytes: make([]byte, 0, presize),
			}
		},
	}
	return func(srv any, stream grpc.ServerStream) error {
		frame := rawFramePool.Get().(*RawFrame)
		frame.RawBytes = frame.RawBytes[:0]
		defer rawFramePool.Put(frame)

		if err := stream.RecvMsg(frame); err != nil {
			if err == io.EOF {
				return nil
			}

			p.logger.Error("failed to receive raw frame", "err", err)
			return status.Errorf(codes.Internal, "failed to read stream")
		}

		headers := map[string][]string{
			"Content-Type": {"application/x-protobuf"},
		}

		if compression := detectCompression(frame.RawBytes); compression != "" {
			headers["Content-Encoding"] = []string{compression}
		}

		if err := p.producer.PublishLogs(stream.Context(), p.natsSubjectPrefix+"raw", frame.RawBytes, headers); err != nil {
			p.logger.Error("failed to publish to nats", "err", err)
			return status.Errorf(codes.Internal, "failed to publish to nats")
		}

		// Unary gRPC contract, client sends one message, server must send exactly one back.
		// Then server closes the connection with status OK.
		return stream.SendMsg(&RawFrame{RawBytes: []byte{}})
	}
}

func (p *proxyHandler) StreamHandler(srv any, stream grpc.ServerStream) error {
	frame := &RawFrame{}

	if err := stream.RecvMsg(frame); err != nil {
		if err == io.EOF {
			return nil
		}

		p.logger.Error("failed to receive raw frame", "err", err)
		return status.Errorf(codes.Internal, "failed to read stream")
	}

	headers := map[string][]string{
		"Content-Type": {"application/x-protobuf"},
	}

	if compression := detectCompression(frame.RawBytes); compression != "" {
		headers["Content-Encoding"] = []string{compression}
	}

	if err := p.producer.PublishLogs(stream.Context(), p.natsSubjectPrefix+"raw", frame.RawBytes, headers); err != nil {
		p.logger.Error("failed to publish to nats", "err", err)
		return status.Errorf(codes.Internal, "failed to publish to nats")
	}

	// Unary gRPC contract, client sends one message, server must send exactly one back.
	// Then server closes the connection with status OK.
	return stream.SendMsg(&RawFrame{RawBytes: []byte{}})
}

// Detect gzip and zstd compression via byte sniffing else it defaults to no compression.
// This is due to grpc not exposing internal encoding information of incoming payloads.
// Since grpc automatically filters out unsupported encodings via only registered
// compressors, we can safely assume what we receive is gzip, zstd, or fallback to
// identity. This is similar with what we do for header checking on the http side.
func detectCompression(b []byte) string {
	if len(b) >= 4 &&
		b[0] == 0x28 &&
		b[1] == 0xB5 &&
		b[2] == 0x2F &&
		b[3] == 0xFD {
		return "zstd"
	}

	if len(b) >= 2 &&
		b[0] == 0x1F &&
		b[1] == 0x8B {
		return "gzip"
	}

	return ""
}

type noopCompressor struct {
	name string
}

func (c *noopCompressor) Name() string {
	return c.name
}

// no compression for outbound traffic
// Not needed here because we are ingesting data
func (c *noopCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	return &noopWriteCloser{w}, nil
}

// returns the original reader, skips decompression
func (c *noopCompressor) Decompress(r io.Reader) (io.Reader, error) {
	return r, nil
}

// wraps an io.Writer to satisfy io.WriteCloser interface
type noopWriteCloser struct {
	io.Writer // inherit Write method
}

func (n *noopWriteCloser) Close() error {
	return nil
}
