package attack

import (
	"fmt"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
)

const codecName = "raw-bytes"

func init() {
	encoding.RegisterCodecV2(&rawCodec{})
}

type rawFrame struct {
	RawBytes []byte
}

type rawCodec struct{}

func (r *rawCodec) Name() string {
	return codecName
}

// only supports rawFrame type
func (r *rawCodec) Marshal(v any) (mem.BufferSlice, error) {
	if out, ok := v.(*rawFrame); ok {
		return mem.BufferSlice{mem.SliceBuffer(out.RawBytes)}, nil
	}

	return nil, fmt.Errorf("unsupported type %T", v)
}

// only supports rawFrame type
func (r *rawCodec) Unmarshal(data mem.BufferSlice, v any) error {
	if out, ok := v.(*rawFrame); ok {
		srcBytes := data.Materialize()
		out.RawBytes = make([]byte, len(srcBytes))
		copy(out.RawBytes, srcBytes)
		return nil
	}

	return fmt.Errorf("unsupported type %T", v)
}
