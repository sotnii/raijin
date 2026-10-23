package raijin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/sotnii/raijin/internal/controllers"
	"github.com/sotnii/raijin/internal/runtime"
	"github.com/sotnii/raijin/internal/runtime/containers"
	"github.com/sotnii/raijin/internal/util"
	"github.com/sotnii/raijin/logging"
)

// TODO: Test running
type Test struct {
	id          string
	name        string
	clusterSpec ClusterSpec
	logger      *slog.Logger
	executor    ExecutorFactory
	runtime     runtime.Runtime
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
	}
	for _, opt := range opts {
		opt(t)
	}
	t.runtime = containers.NewContainersRuntime(t.logger)

	return t
}

func (t *Test) executorDeps() TestExecutorDeps {
	// TODO: introduce node controller factory
	nc := controllers.NewNodeController(t.runtime.NodeFactory(), t.logger.With("component", "node_controller"))

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
		for line := range strings.SplitSeq(testError.Err.Error(), "\n") {
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
