package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type (
	cmd struct {
		Label     string
		Cmd       *exec.Cmd
		StartTime time.Time
		EndTime   time.Time
	}
	Process struct {
		Name   string
		Stdout *bytes.Buffer
		Stderr *bytes.Buffer

		cmd  []*cmd
		err  error
		done chan struct{}
	}
)

func NewProcess(name string) *Process {
	return &Process{
		Name:   name,
		done:   make(chan struct{}),
		Stdout: bytes.NewBuffer(nil),
		Stderr: bytes.NewBuffer(nil),
	}
}

func (p *Process) WithCommand(c *exec.Cmd) *Process {
	p.cmd = append(p.cmd, &cmd{Cmd: c})
	return p
}

func (p *Process) WithLabeledCommand(c *exec.Cmd, label string) *Process {
	p.cmd = append(p.cmd, &cmd{Cmd: c, Label: label})
	return p
}

func (p *Process) Start(ctx context.Context) error {
	if len(p.cmd) == 0 {
		close(p.done)
		return nil
	}
	for _, c := range p.cmd {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.Cmd.Start(); err != nil {
				return fmt.Errorf("failed to start command %s: %w", c.prefix(p), err)
			}
			c.StartTime = time.Now()
		}
	}

	go func() {
		defer close(p.done)
		for _, c := range p.cmd {
			out, err := c.Cmd.Output()
			for _, os := range strings.Split(string(out), "\n") {
				p.Stdout.WriteString(fmt.Sprintf("[%s] %s\n", c.prefix(p), os))
			}
			c.EndTime = time.Now()
			if err != nil {
				var perr *exec.ExitError
				if errors.As(err, &perr) {
					p.err = fmt.Errorf("process %s failed with code %d", c.prefix(p), perr.ExitCode())
					for _, es := range strings.Split(string(perr.Stderr), "\n") {
						if es != "" {
							p.Stderr.WriteString(fmt.Sprintf("[%s] %s\n", c.prefix(p), es))
						}
					}
				} else {
					p.err = fmt.Errorf("process %s failed: %w", c.prefix(p), err)
				}
				return
			}
		}
	}()

	return nil
}

func (p *Process) Done() <-chan struct{} {
	return p.done
}

func (p *Process) Error() error {
	<-p.Done()
	return p.err
}

func (p *Process) StartTime() time.Time {
	if len(p.cmd) == 0 {
		return time.Time{}
	}
	return p.cmd[0].StartTime
}

func (p *Process) EndTime() time.Time {
	<-p.Done()
	if len(p.cmd) == 0 {
		return time.Time{}
	}
	return p.cmd[len(p.cmd)-1].EndTime
}

func (p *Process) ElapsedTime() time.Duration {
	select {
	case <-p.Done():
		return p.EndTime().Sub(p.StartTime())
	default:
		return time.Since(p.StartTime())
	}
}

func (c *cmd) prefix(p *Process) string {
	if c.Label == "" {
		return p.Name
	}

	return fmt.Sprintf("%s:%s", p.Name, c.Label)
}
