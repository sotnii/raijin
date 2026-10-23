package containers

import (
	"log/slog"

	"github.com/sotnii/raijin/internal/drivers/execution"
	"github.com/sotnii/raijin/internal/drivers/network/namespace"
	"github.com/sotnii/raijin/internal/runtime"
)

type ContainerRuntime struct {
	factory *ContainerNodeFactory
}

// NewContainersRuntime returns a runtime that will use host's network stack to manage network (namespaces, routing) and
// containerd to run services on the node.
func NewContainersRuntime(logger *slog.Logger) *ContainerRuntime {
	factory := newContainerNodeFactory(
		namespace.NewNamespaceNetworkDriver("rjn", logger.With("component", "ip_cmd_driver"), nil),
		execution.NewContainerdExecutionDriver(),
	)
	return &ContainerRuntime{
		factory: &factory,
	}
}

func (r *ContainerRuntime) NodeFactory() runtime.NodeFactory {
	return r.factory
}
