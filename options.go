package raijin

import (
	"log/slog"

	"github.com/sotnii/raijin/spec"
)

type ClusterSpec = spec.ClusterSpec

type RuntimeFactory func(name string, cluster spec.ClusterSpec, logger *slog.Logger) (*TestRuntime, error)

type TestOption func(*Test)

func WithLogger(logger *slog.Logger) TestOption {
	return func(t *Test) {
		if logger != nil {
			t.logger = logger
		}
	}
}
