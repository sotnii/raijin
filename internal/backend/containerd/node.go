package containerd

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sotnii/raijin/spec"
)

type state int

const (
	new         state = iota
	provisioned state = iota
	torndown    state = iota
	failed      state = iota
)

func (s state) String() string {
	switch s {
	case new:
		return "new"
	case provisioned:
		return "provisioned"
	case torndown:
		return "torn-down"
	case failed:
		return "failed"
	default:
		return "uknown"
	}
}

type NetworkDriver interface {
	ProvisionNamespace(ctx context.Context, name string) error
	TeardownNamespace(ctx context.Context, name string) error
}

type ContainerDriver interface{}

type ContainerdNode struct {
	spec spec.NodeSpec

	logger     *slog.Logger
	network    NetworkDriver
	containerd ContainerDriver

	mu sync.Mutex
	// TODO: test state transitions and validations
	state state
}

func NewContainerdNode(spec spec.NodeSpec, network NetworkDriver, containerd ContainerDriver, logger *slog.Logger) *ContainerdNode {
	return &ContainerdNode{
		spec:       spec,
		logger:     logger.With("node", spec.ID),
		state:      new,
		network:    network,
		containerd: containerd,
		// TODO:
		// network: ,
		// containerd: ,
	}
}

func (n *ContainerdNode) Provision(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != new {
		return fmt.Errorf("can only provision a node in a new state, got: %s", n.state)
	}

	n.logger.Debug("provisioning node")

	n.state = provisioned
	// return n.network.ProvisionNamespace(ctx, string(n.spec.ID))
	return nil
}

func (n *ContainerdNode) Teardown(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state == torndown {
		return nil
	}

	n.logger.Debug("tearing down node")

	n.state = torndown
	return nil
}
