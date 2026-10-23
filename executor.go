package raijin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/sotnii/raijin/spec"
)

type NodeController interface {
	ProvisionNode(context.Context, spec.NodeSpec) error
	TeardownNode(context.Context, spec.NodeID) error
}

type TestExecutorDeps struct {
	NodeController NodeController
}

type TestExecutor struct {
	name   string
	spec   spec.ClusterSpec
	logger *slog.Logger

	nodes NodeController
	// containers   managers.ContainerManager
	// workDir      string
	// artifactsDir string

	cleanups []namedCleanup
}

type signalState struct {
	mu  sync.Mutex
	sig os.Signal
}

type CleanupFunc func(context.Context) error

type namedCleanup struct {
	name string
	fn   CleanupFunc
}

func (c namedCleanup) run(
	ctx context.Context,
	logger *slog.Logger,
) (err error) {
	defer func() {
		if v := recover(); v != nil {
			logger.Error(
				"cleanup panicked",
				"cleanup", c.name,
				"panic", v,
				"stack", string(debug.Stack()),
			)
			err = fmt.Errorf("panicked: %v", v)
		}
	}()

	return c.fn(ctx)
}

func (e *TestExecutor) registerCtrlCListener(ctx context.Context, cancel context.CancelFunc) *signalState {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	sigState := &signalState{}
	go func() {
		defer signal.Stop(signals)
		select {
		case sig := <-signals:
			sigState.set(sig)
			e.logger.Warn("received termination signal", "signal", sig.String())
			cancel()
		case <-ctx.Done():
		}
	}()
	return sigState
}

func (s *signalState) set(sig os.Signal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sig == nil {
		s.sig = sig
	}
}

func (s *signalState) get() os.Signal {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sig
}

func NewTestExecutor(
	name string,
	cluster spec.ClusterSpec,
	deps TestExecutorDeps,
	logger *slog.Logger,
) (*TestExecutor, error) {
	// workDir := filepath.Join(os.TempDir(), "raijin")
	// if err := os.MkdirAll(workDir, 0o755); err != nil {
	// 	return nil, fmt.Errorf("create work dir: %w", err)
	// }

	// nsMgr, err := network.NewManager("rjn", network.ExecCommander{}, logger.With("component", "network"))
	// if err != nil {
	// 	return nil, err
	// }

	// containerRuntimeManager, err := containers.NewRuntimeManager(containers.Config{
	// 	Socket:      containers.DefaultSocket,
	// 	Namespace:   containers.DefaultNamespace,
	// 	Snapshotter: containers.DefaultSnapshotter,
	// 	WorkDir:     workDir,
	// 	Logger:      logger.With("component", "container_runtime"),
	// })
	// if err != nil {
	// 	return nil, err
	// }

	return &TestExecutor{
		name:   name,
		spec:   cluster,
		logger: logger,
		nodes:  deps.NodeController,

		// containers:   managers.NewContainerManager(containerRuntimeManager, logger),
		// network:      managers.NewNetworkManager(nsMgr, logger),
		// workDir:      workDir,
		// artifactsDir: filepath.Join(".raijin", fmt.Sprintf("%s-%s", name, testId)),
	}, nil
}

func (e *TestExecutor) Defer(name string, fn CleanupFunc) {
	e.cleanups = append(e.cleanups, namedCleanup{name, fn})
}

func (e *TestExecutor) Run(ctx context.Context, fn func(*TestHandle) error) *TestResult {
	result := &TestResult{}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	e.registerCtrlCListener(ctx, cancel)

	defer e.finalize(result, cancel)

	if err := e.provision(ctx); err != nil {
		result.Add(err)
		return result
	}

	return result
}

func (e *TestExecutor) provision(ctx context.Context) error {
	e.logger.Info("provisioning test environment")

	for _, v := range e.spec.Nodes {
		e.Defer(
			fmt.Sprintf("teardown %v", v.ID),
			func(ctx context.Context) error { return e.nodes.TeardownNode(ctx, v.ID) },
		)
		err := e.nodes.ProvisionNode(ctx, v)
		if err != nil {
			return err
		}
	}

	e.logger.Info("runtime environment ready")
	return nil
}

func (e *TestExecutor) Cleanup(ctx context.Context) error {
	// Consume the stack so another Cleanup call won't replay it.
	cleanups := e.cleanups
	e.cleanups = nil

	var errs []error
	for _, cleanup := range slices.Backward(cleanups) {
		if err := cleanup.run(ctx, e.logger); err != nil {
			errs = append(errs, fmt.Errorf(
				"cleanup %s: %w", cleanup.name, err,
			))
		}
	}

	return errors.Join(errs...)
}

func (e *TestExecutor) finalize(result *TestResult, cancel context.CancelFunc) {
	if v := recover(); v != nil {
		result.AddWithStackTrace(
			fmt.Errorf("test runtime panicked: %v", v),
			string(debug.Stack()),
		)
	}

	cancel()

	ctx, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()

	result.Add(e.Cleanup(ctx))
}

// func (r *TestRuntime) runTest(ctx context.Context, cancel context.CancelFunc, sigState *signalState, handle *TestHandle, fn func(*TestHandle) error) TestResult {
// 	events, errs, err := r.containers.ObserveEvents(ctx)
// 	if err != nil {
// 		r.logger.Warn("container event subscription failed", "error", err)
// 	}

// 	testResult := make(chan TestResult, 1)
// 	go func() {
// 		defer func() {
// 			if v := recover(); v != nil {
// 				testResult <- TestResult{
// 					ErrorMessage: fmt.Sprintf("test panicked: %v", v),
// 					StackTrace:   string(debug.Stack()),
// 				}
// 			}
// 		}()

// 		r.logger.Info(fmt.Sprintf("starting test %q", r.name))
// 		if err := fn(handle); err != nil {
// 			testResult <- TestResult{ErrorMessage: err.Error()}
// 			return
// 		}
// 		testResult <- TestResult{}
// 	}()

// 	var testFinished bool

// 	for {
// 		select {
// 		case result := <-testResult:
// 			testFinished = true
// 			r.logger.Debug("user test callback finished", "error", result.ErrorMessage)
// 			cancel()
// 			return result
// 		case event, ok := <-events:
// 			if ok {
// 				r.logger.Warn("container process exited", "container_id", event.ContainerID, "exec_id", event.ExecID, "exit_code", event.ExitCode)
// 			} else {
// 				events = nil
// 			}
// 		case err, ok := <-errs:
// 			if ok && err != nil {
// 				r.logger.Warn("container event stream failed", "error", err)
// 			} else {
// 				errs = nil
// 			}
// 		case <-ctx.Done():
// 			if testFinished {
// 				r.logger.Debug("runtime context canceled after normal completion")
// 				return TestResult{}
// 			}
// 			return resultFromError(ctx.Err(), sigState)
// 		}
// 	}
// }

// func resultFromError(err error, sigState *signalState) TestResult {
// 	if err == nil {
// 		return TestResult{}
// 	}
// 	if sig := sigState.get(); sig != nil {
// 		return TestResult{ErrorMessage: fmt.Errorf("%w: %s", runtime.ErrTestCancelled, sig.String()).Error()}
// 	}
// 	return TestResult{ErrorMessage: err.Error()}
// }

// func (r *TestRuntime) cleanupHandle(handle *TestHandle) (err error) {
// 	if handle == nil {
// 		return nil
// 	}

// 	defer func() {
// 		if v := recover(); v != nil {
// 			r.logger.Error("api handle cleanup panicked", "panic", v, "stack", string(debug.Stack()))
// 			err = fmt.Errorf("api handle cleanup panicked: %v", v)
// 		}
// 	}()

// 	r.logger.Debug("cleaning up api handles")
// 	return handle.Cleanup()
// }

// func (r *TestRuntime) teardown(ctx context.Context) error {
// 	var errs []error

// 	for _, node := range r.spec.Nodes {
// 		for _, container := range r.containers.ContainersOf(node.ID) {
// 			if err := r.collectContainerArtifacts(node.ID, container); err != nil {
// 				r.logger.Error("artifact collection failed", "node", node.ID, "service", container.Name, "error", err)
// 				errs = append(errs, err)
// 			}
// 		}
// 	}
// 	r.logger.Info("artifacts collected", "destination", r.artifactsDir)

// 	r.logger.Info("tearing down runtime")
// 	err := r.containers.Teardown(ctx)
// 	if err != nil {
// 		errs = append(errs, err)
// 	}
// 	r.logger.Info("containers tore down")

// 	err = r.network.Teardown(ctx)
// 	if err != nil {
// 		errs = append(errs, err)
// 	}
// 	r.logger.Info("network tore down")

// 	return errors.Join(errs...)
// }

// func (r *TestRuntime) collectContainerArtifacts(nodeID spec.NodeID, ctr containers.RunningContainer) error {
// 	dstDir := filepath.Join(r.artifactsDir, "logs", string(nodeID), ctr.Name)
// 	r.logger.Debug("collecting container artifacts", "node", nodeID, "service", ctr.Name, "src", ctr.IO.Dir, "dst", dstDir)
// 	if err := os.MkdirAll(dstDir, 0o755); err != nil {
// 		return fmt.Errorf("create artifact dir for %s/%s: %w", nodeID, ctr.Name, err)
// 	}

// 	if err := util.CopyFile(ctr.IO.Stdout, filepath.Join(dstDir, "stdout")); err != nil {
// 		return fmt.Errorf("copy stdout for %s/%s: %w", nodeID, ctr.Name, err)
// 	}
// 	if err := util.CopyFile(ctr.IO.Stderr, filepath.Join(dstDir, "stderr")); err != nil {
// 		return fmt.Errorf("copy stderr for %s/%s: %w", nodeID, ctr.Name, err)
// 	}
// 	return nil
// }
