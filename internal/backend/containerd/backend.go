package containerd

import "github.com/sotnii/raijin/internal/backend"

type ContainerdOnHostBackend struct {
	factory *ContainerdNodeFactory
}

// NewContainerdOnHostBackend returns a runtime backend that will use host's network stack to manage network (namespaces, routing) and
// containerd to run services on the node.
func NewContainerdOnHostBackend() *ContainerdOnHostBackend {
	factory := NewContainerdNodeFactory()
	return &ContainerdOnHostBackend{
		factory: &factory,
	}
}

func (b *ContainerdOnHostBackend) NodeFactory() backend.NodeFactory {
	return b.factory
}
