package bazel

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	libio "github.com/edevhub/bazel-routines/pkg/io"
)

type (
	// QueryOutput defines an object to set up and read specific query output format
	QueryOutput interface {
		Format() string
		Read(io.Reader) error
	}

	LabelOutput struct {
		Labels []string
	}

	XmlOutput struct {
		Result interface{}
	}

	QueryResult struct {
		Err    error
		stderr io.Reader
	}
)

func (l *LabelOutput) Format() string {
	return "label"
}

func (l *LabelOutput) Read(reader io.Reader) (err error) {
	l.Labels, err = libio.ReadLines(reader, strings.TrimSpace)
	return
}

func (o *XmlOutput) Format() string {
	return "xml"
}

func (o *XmlOutput) Read(reader io.Reader) (err error) {
	if o.Result == nil {
		return errors.New("output Result cannot be nil")
	}

	lines, err := libio.ReadLines(reader, func(s string) string {
		return s
	})
	if err != nil {
		return
	}

	// We cut xml header line, since it is version 1.1 and go support is on only version 1.0
	// without the header we still are able to parse xml content normally
	data := []byte(strings.Join(lines[1:], "\n"))

	if err = xml.Unmarshal(data, &o.Result); err != nil {
		return fmt.Errorf("failed to unmarshal xml query result: %w", err)
	}
	return nil
}

func (r QueryResult) ErrorLines() ([]string, error) {
	return libio.ReadLines(r.stderr, strings.TrimSpace)
}

func (c *Client) Query(ctx context.Context, q string, output QueryOutput) (r QueryResult) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	args := []string{
		"query",
		fmt.Sprintf("--output=%s", output.Format()),
		q,
	}

	cmd := c.cmd(ctx, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	r.stderr = stderr

	if err := cmd.Run(); err != nil {
		r.Err = fmt.Errorf("failed to execute bazel query command: %w", err)
		return
	}

	if err := output.Read(stdout); err != nil {
		r.Err = fmt.Errorf("failed to read bazel query output: %w", err)
		return
	}

	return
}
