package attack

import (
	"github.com/Eddrick-23/Logarithm/load_generator/config"
	"github.com/Eddrick-23/Logarithm/load_generator/runner"
)

func NewRunner(cfg *config.CleanConfig, outDir string) (runner.Runner, error) {
	factory, err := NewPayloadFactory(cfg)
	if err != nil {
		return nil, err
	}

	// TODO switch statement for grpc and http protocols once config added
	httpRunner, err := newHttpRunner(cfg, factory, outDir)
	if err != nil {
		return nil, err
	}

	return httpRunner, nil
}
