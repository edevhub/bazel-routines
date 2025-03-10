package bazel

import (
	"context"
	"fmt"
	"os"
	goexec "os/exec"

	"github.com/edevhub/bazel-routines/pkg/exec"
)

// Run will prepare provided target with optional extra args as ready to start process
//
//	When args provided it will be passed forward to the executable under the target,
//	for example: bazel run //:sample-target -- extra args
//
// Run prepares bazel run spawn with --script_path option
// [see: https://docs.bazel.build/versions/main/command-line-reference.html#flag--script_path].
//
//		Instead of running executable target on bazel server directly, it is saved as a standalone runnable bash script,
//	 this allows tracking of build phase and run phases separately and free up some resources from bazel process
func (c *Client) Run(ctx context.Context, target string, args ...string) (*exec.Process, error) {
	script, err := func() (string, error) {
		f, err := os.CreateTemp("", "run-target-*")
		if err != nil {
			return "", err
		}

		defer f.Close()

		if err = f.Chmod(0o700); err != nil {
			return "", err
		}

		return f.Name(), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to create temp executable file: %w", err)
	}

	buildArgs := []string{"run", "--noshow_progress", fmt.Sprintf("--script_path=%s", script), target}
	buildCmd := c.cmd(ctx, buildArgs...)

	runCmd := goexec.CommandContext(ctx, script, args...)
	runCmd.Dir = c.workspace

	p := exec.NewProcess(target)
	p.WithLabeledCommand(buildCmd, "build")
	p.WithLabeledCommand(runCmd, "run")

	return p, nil
}
