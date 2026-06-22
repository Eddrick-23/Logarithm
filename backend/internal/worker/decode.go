package worker

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

type Decoder interface {
	decode(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error)
}

type LogDecoder struct {
}

func NewLogDecoder() *LogDecoder {
	return &LogDecoder{}
}

func (d *LogDecoder) decode(payload []byte, headers map[string][]string) (*plogotlp.ExportRequest, error) {
	req := plogotlp.NewExportRequest()

	contentType := ""
	if vals, ok := headers["Content-Type"]; ok && len(vals) > 0 {
		contentType = vals[0]
	}

	if strings.HasPrefix(contentType, "application/x-protobuf") {
		if err := req.UnmarshalProto(payload); err != nil {
			return nil, err
		}
		return &req, nil
	}

	if strings.HasPrefix(contentType, "application/json") {
		if err := req.UnmarshalJSON(payload); err != nil {
			return nil, err
		}
		return &req, nil
	}

	return nil, fmt.Errorf("unsupported or missing content type: %s", contentType)
}
