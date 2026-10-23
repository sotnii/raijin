package containers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/sotnii/raijin/internal/drivers/network/namespace"
	"github.com/sotnii/raijin/internal/runtime/containers"
	"github.com/sotnii/raijin/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type networkDriverFake struct {
	handle         *namespace.NamespaceHandle
	provisionCalls int
	provisionErr   error
	teardownCalls  []namespace.NamespaceHandle
	teardownErr    error
}

func (n *networkDriverFake) ProvisionNamespace(context.Context) (*namespace.NamespaceHandle, error) {
	n.provisionCalls++
	if n.provisionErr != nil {
		// A failed acquisition leaves no resource owned by the node.
		return nil, n.provisionErr
	}
	return n.handle, nil
}

func (n *networkDriverFake) TeardownNamespace(_ context.Context, handle namespace.NamespaceHandle) error {
	n.teardownCalls = append(n.teardownCalls, handle)
	return n.teardownErr
}

// Construct an opaque namespace handle without changing host networking.
type namespaceCommanderFake struct{}

func (namespaceCommanderFake) Run(context.Context, string, ...string) (string, error) {
	return "", nil
}

type nodeFixture struct {
	node    *containers.ContainerNode
	network *networkDriverFake
}

func newNodeFixture(t *testing.T) *nodeFixture {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	driver := namespace.NewNamespaceNetworkDriver("node-test", logger, namespaceCommanderFake{})
	handle, err := driver.ProvisionNamespace(t.Context())
	require.NoError(t, err)
	require.NotNil(t, handle)

	network := &networkDriverFake{handle: handle}
	return &nodeFixture{
		node:    containers.NewContainerNode(spec.NodeSpec{ID: "test"}, network, nil, logger),
		network: network,
	}
}

func TestProvisionRejectsAlreadyProvisionedNode(t *testing.T) {
	f := newNodeFixture(t)
	require.NoError(t, f.node.Provision(t.Context()))

	err := f.node.Provision(t.Context())
	assert.ErrorContains(t, err, "got: provisioned")
	assert.Equal(t, 1, f.network.provisionCalls)
}

func TestTeardownReleasesAcquiredNamespace(t *testing.T) {
	f := newNodeFixture(t)
	require.NoError(t, f.node.Provision(t.Context()))

	require.NoError(t, f.node.Teardown(t.Context()))
	assert.Equal(t, []namespace.NamespaceHandle{*f.network.handle}, f.network.teardownCalls)
}

func TestTeardownIsIdempotent(t *testing.T) {
	f := newNodeFixture(t)
	require.NoError(t, f.node.Provision(t.Context()))
	require.NoError(t, f.node.Teardown(t.Context()))

	require.NoError(t, f.node.Teardown(t.Context()))
	assert.Len(t, f.network.teardownCalls, 1)
}

func TestProvisionRejectsTornDownNode(t *testing.T) {
	f := newNodeFixture(t)
	require.NoError(t, f.node.Provision(t.Context()))
	require.NoError(t, f.node.Teardown(t.Context()))

	err := f.node.Provision(t.Context())
	assert.ErrorContains(t, err, "got: torndown")
	assert.Equal(t, 1, f.network.provisionCalls)
}

func TestTeardownErrorCanBeRetried(t *testing.T) {
	f := newNodeFixture(t)
	require.NoError(t, f.node.Provision(t.Context()))
	failure := errors.New("namespace deletion failed")
	f.network.teardownErr = failure

	err := f.node.Teardown(t.Context())
	assert.ErrorIs(t, err, failure)
	assert.ErrorContains(t, err, "failed to teardown network namespace")

	f.network.teardownErr = nil
	require.NoError(t, f.node.Teardown(t.Context()))
	assert.Equal(t, []namespace.NamespaceHandle{
		*f.network.handle,
		*f.network.handle,
	}, f.network.teardownCalls)

	require.NoError(t, f.node.Teardown(t.Context()))
	assert.Len(t, f.network.teardownCalls, 2)
}

func TestProvisionFailureRejectsRetryAndNeedsNoNamespaceCleanup(t *testing.T) {
	f := newNodeFixture(t)
	failure := errors.New("namespace creation failed")
	f.network.provisionErr = failure

	err := f.node.Provision(t.Context())
	assert.ErrorIs(t, err, failure)
	assert.ErrorContains(t, err, "failed to provision network namespace")

	err = f.node.Provision(t.Context())
	assert.ErrorContains(t, err, "got: failed")
	assert.Equal(t, 1, f.network.provisionCalls)

	require.NoError(t, f.node.Teardown(t.Context()))
	assert.Empty(t, f.network.teardownCalls)
}

func TestTeardownBeforeProvisionClosesNode(t *testing.T) {
	f := newNodeFixture(t)
	require.NoError(t, f.node.Teardown(t.Context()))
	assert.Empty(t, f.network.teardownCalls)

	err := f.node.Provision(t.Context())
	assert.ErrorContains(t, err, "got: torndown")
	assert.Zero(t, f.network.provisionCalls)
}
