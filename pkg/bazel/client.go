package bazel

import (
	"context"
	"os/exec"

	"github.com/kelseyhightower/envconfig"
)

type ExecError = exec.ExitError

type ClientConfig struct {
	Workspace      string   `envconfig:"BUILD_WORKSPACE_DIRECTORY" required:"true"`
	BinPath        string   `envconfig:"BAZEL_BIN_PATH" default:"bazel"`
	StartupOptions []string `envconfig:"BAZEL_STARTUP_OPTIONS"`
}

type Client struct {
	bazel          string
	startupOptions []string
	workspace      string
}

func NewClient(cfg *ClientConfig) *Client {
	return &Client{
		bazel:          cfg.BinPath,
		startupOptions: cfg.StartupOptions,
		workspace:      cfg.Workspace,
	}
}

func (c *Client) Workspace() string {
	return c.workspace
}

func (c *Client) cmd(ctx context.Context, args ...string) *exec.Cmd {
	withStartupArgs := append(c.startupOptions, args...)
	cmd := exec.CommandContext(ctx, c.bazel, withStartupArgs...)
	cmd.Dir = c.workspace

	return cmd
}

func (c *ClientConfig) Load() error {
	return envconfig.Process("", c)
}
