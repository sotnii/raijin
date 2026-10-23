package containerd

import (
	"context"
	"log/slog"

	"github.com/sotnii/raijin/internal/backend/node"
	"github.com/sotnii/raijin/spec"
)

type ContainerdNodeFactory struct {
}

func NewContainerdNodeFactory() ContainerdNodeFactory {
	return ContainerdNodeFactory{}
}

func (f *ContainerdNodeFactory) NewNode(_ context.Context, spec spec.NodeSpec, logger *slog.Logger) node.Node {
	return NewContainerdNode(spec, nil, nil, logger)
}
