package backend

import (
	"context"
	"log/slog"

	"github.com/sotnii/raijin/internal/backend/node"
	"github.com/sotnii/raijin/spec"
)

type NodeFactory interface {
	NewNode(context.Context, spec.NodeSpec, *slog.Logger) node.Node
}

type RuntimeBackend interface {
	NodeFactory() NodeFactory
}
