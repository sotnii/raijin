package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sotnii/raijin/internal/backend"
	"github.com/sotnii/raijin/internal/backend/node"
	"github.com/sotnii/raijin/spec"
)

type NodeController struct {
	logger      *slog.Logger
	nodeFactory backend.NodeFactory

	nodes map[spec.NodeID]node.Node

	mu sync.Mutex
}

func NewNodeController(factory backend.NodeFactory, logger *slog.Logger) *NodeController {
	return &NodeController{
		logger:      logger,
		nodes:       make(map[spec.NodeID]node.Node),
		nodeFactory: factory,
	}
}

func (nc *NodeController) ProvisionNode(ctx context.Context, nodeSpec spec.NodeSpec) error {
	node := nc.nodeFactory.NewNode(ctx, nodeSpec, nc.logger)

	nc.mu.Lock()
	defer nc.mu.Unlock()

	if _, ok := nc.nodes[nodeSpec.ID]; ok {
		return errors.New("nodes must be torn down before provisoining")
	}
	nc.nodes[nodeSpec.ID] = node

	err := node.Provision(ctx)
	if err != nil {
		return fmt.Errorf("failed to provision node network: %w", err)
	}

	return nil
}

func (nc *NodeController) TeardownNode(ctx context.Context, id spec.NodeID) error {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	node, ok := nc.nodes[id]
	if !ok {
		return nil
	}

	err := node.Teardown(ctx)
	if err != nil {
		return fmt.Errorf("failed network teardown for node %s: %w", id, err)
	}

	delete(nc.nodes, id)

	return nil
}
