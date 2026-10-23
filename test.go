package raijin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/sotnii/raijin/internal/backend"
	"github.com/sotnii/raijin/internal/backend/containerd"
	"github.com/sotnii/raijin/internal/controllers"
	"github.com/sotnii/raijin/internal/runtime/util"
	"github.com/sotnii/raijin/logging"
)

type Test struct {
	id          string
	name        string
	clusterSpec ClusterSpec
	logger      *slog.Logger
	executor    RuntimeFactory
	backend     backend.RuntimeBackend
}

func NewTest(name string, cluster *ClusterSpec, opts ...TestOption) *Test {
	if cluster == nil {
		panic("cluster spec cannot be nil")
	}

	id := util.NewResourceID()
	t := &Test{
		id:          id,
		name:        name,
		clusterSpec: *cluster,
		logger:      logging.NewDefaultLogger(),
		executor:    NewTestExecutor,
		backend:     containerd.NewContainerdOnHostBackend(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *Test) executorDeps() TestExecutorDeps {
	nc := controllers.NewNodeController(t.backend.NodeFactory(), t.logger.With("component", "node_controller"))

	return TestExecutorDeps{
		NodeController: nc,
	}
}

func (t *Test) Run(ctx context.Context, fn func(*TestHandle) error) {
	deps := t.executorDeps()
	e, err := t.executor(t.name, t.clusterSpec, deps, t.logger)
	if err != nil {
		t.logger.Error("error creating runtime", "error", err)
		panic(err)
	}
	result := e.Run(ctx, fn)
	if result.Failed() {
		t.logger.Error("test failed", "errors", len(result.Errors))
		fmt.Fprintln(os.Stderr, prettyTestErrors(result.Errors))
		os.Exit(1)
	}
	t.logger.Info("test passed")
}

func prettyTestErrors(testErrors []TestError) string {
	var b strings.Builder
	for i, testError := range testErrors {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "Error %d:\n", i+1)
		for _, line := range strings.Split(testError.Err.Error(), "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
		if trace := prettyStackTrace(testError.StackTrace); trace != "" {
			b.WriteByte('\n')
			b.WriteString(trace)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func prettyStackTrace(stackTrace string) string {
	lines := strings.Split(strings.TrimSpace(stackTrace), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("panic stack trace:\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
