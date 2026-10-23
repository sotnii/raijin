package runtime

import (
	"context"
	"log/slog"

	"github.com/sotnii/raijin/internal/runtime/node"
	"github.com/sotnii/raijin/spec"
)

type NodeFactory interface {
	NewNode(context.Context, spec.NodeSpec, *slog.Logger) node.Node
}

type Runtime interface {
	NodeFactory() NodeFactory
}
