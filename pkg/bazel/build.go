package bazel

import (
	"context"
	"errors"
	"fmt"
	"github.com/edevhub/bazel-routines/pkg/exec"
)

// Build will prepare bazel build SingleProcess ready to be started
func (c *Client) Build(ctx context.Context, targets ...string) (*exec.Process, error) {
	totalTargets := len(targets)
	if totalTargets == 0 {
		return nil, errors.New("at least one target must be provided")
	}

	cmd := c.cmd(ctx, append([]string{"build", "--noshow_progress"}, targets...)...)
	label := fmt.Sprintf("build %s", targets[0])
	if totalTargets > 1 {
		label = fmt.Sprintf("%d target(s)", totalTargets)
	}

	p := exec.NewProcess(label)
	p.WithCommand(cmd)
	return p, nil
}
