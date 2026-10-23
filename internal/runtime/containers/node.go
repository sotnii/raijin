package containers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sotnii/raijin/internal/drivers/network/namespace"
	"github.com/sotnii/raijin/spec"
)

type nodeState int

const (
	new         nodeState = iota
	provisioned nodeState = iota
	torndown    nodeState = iota
	failed      nodeState = iota
)

func (s nodeState) String() string {
	switch s {
	case new:
		return "new"
	case provisioned:
		return "provisioned"
	case torndown:
		return "torndown"
	case failed:
		return "failed"
	default:
		return "uknown"
	}
}

type NetworkDriver interface {
	ProvisionNamespace(context.Context) (*namespace.NamespaceHandle, error)
	TeardownNamespace(context.Context, namespace.NamespaceHandle) error
}

type ContainerExecutionDriver interface{}

type ContainerNode struct {
	spec spec.NodeSpec

	logger     *slog.Logger
	network    NetworkDriver
	containers ContainerExecutionDriver

	mu    sync.Mutex
	state nodeState
	ns    *namespace.NamespaceHandle
}

func NewContainerNode(spec spec.NodeSpec, network NetworkDriver, containers ContainerExecutionDriver, logger *slog.Logger) *ContainerNode {
	return &ContainerNode{
		spec:       spec,
		logger:     logger.With("node", spec.ID),
		state:      new,
		network:    network,
		containers: containers,
	}
}

func (n *ContainerNode) Provision(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != new {
		return fmt.Errorf("can only provision a node in a new state, got: %s", n.state)
	}

	n.logger.Debug("provisioning node")

	ns, err := n.network.ProvisionNamespace(ctx)
	if err != nil {
		n.state = failed
		return fmt.Errorf("failed to provision network namespace: %w", err)
	}
	n.ns = ns

	n.state = provisioned
	return nil
}

func (n *ContainerNode) Teardown(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state == torndown {
		return nil
	}

	n.logger.Debug("tearing down node")

	if n.ns != nil {
		err := n.network.TeardownNamespace(ctx, *n.ns)
		if err != nil {
			return fmt.Errorf("failed to teardown network namespace: %w", err)
		}
		n.ns = nil
	}

	n.state = torndown
	return nil
}
