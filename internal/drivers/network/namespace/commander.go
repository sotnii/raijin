package namespace

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// todo: put in utils
type execCommander struct{}

func (execCommander) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = err.Error()
		}
		return "", fmt.Errorf("%s %v: %s", name, args, errText)
	}

	return stdout.String(), nil
}
