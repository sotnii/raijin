package namespace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sotnii/raijin/internal/util"
)

type NamespaceHandle struct {
	name, path string
}

type Commander interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type NamespaceNetworkDriver struct {
	prefix string
	cmd    Commander
	logger *slog.Logger
}

// TODO: Test
func NewNamespaceNetworkDriver(prefix string, logger *slog.Logger, cmd Commander) *NamespaceNetworkDriver {
	if cmd == nil {
		cmd = &execCommander{}
	}
	return &NamespaceNetworkDriver{
		cmd:    cmd,
		prefix: prefix,
		logger: logger,
	}
}

func (d *NamespaceNetworkDriver) ProvisionNamespace(ctx context.Context) (*NamespaceHandle, error) {
	id := util.NewResourceID()
	name := fmt.Sprintf("%s-%s", d.prefix, id)
	path := filepath.Join("/run/netns", name)
	d.logger.Debug("creating namespace", "name", id, "path", path)

	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("namespace already exists")
	}

	if _, err := d.cmd.Run(ctx, "ip", "netns", "add", name); err != nil {
		return nil, fmt.Errorf("create namespace %s: %w", name, err)
	}

	return &NamespaceHandle{name: name, path: path}, nil
}

func (d *NamespaceNetworkDriver) TeardownNamespace(ctx context.Context, ns NamespaceHandle) error {
	d.logger.Debug("deleting namespace", "name", ns.name)

	_, err := d.cmd.Run(ctx, "ip", "netns", "del", ns.name)
	if err != nil {
		return fmt.Errorf("delete namespace %s: %w", ns.name, err)
	}
	return nil
}
