package containers

import (
	"context"
	"log/slog"

	"github.com/sotnii/raijin/internal/runtime/node"
	"github.com/sotnii/raijin/spec"
)

type ContainerNodeFactory struct {
	networkDriver   NetworkDriver
	executionDriver ContainerExecutionDriver
}

func newContainerNodeFactory(networkDriver NetworkDriver, executionDriver ContainerExecutionDriver) ContainerNodeFactory {
	return ContainerNodeFactory{
		networkDriver:   networkDriver,
		executionDriver: executionDriver,
	}
}

func (f *ContainerNodeFactory) NewNode(_ context.Context, spec spec.NodeSpec, logger *slog.Logger) node.Node {
	return NewContainerNode(
		spec,
		f.networkDriver,
		f.executionDriver,
		logger,
	)
}
