package attack

import (
	"fmt"

	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/runner"
)

func NewRunner(cfg *config.CleanConfig, outDir string) (runner.Runner, error) {
	factory, err := NewPayloadFactory(cfg)
	if err != nil {
		return nil, err
	}

	switch cfg.Protocol {
	case "http":
		return newHttpRunner(cfg, factory, outDir), nil
	case "grpc":
		return newGRPCRunner(cfg, factory, outDir), nil
	default:
		return nil, fmt.Errorf("unsupported protocol %v", cfg.Protocol)
	}
}
