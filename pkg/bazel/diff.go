package bazel

import (
	"context"
	"fmt"
	"os"
	"strings"

	libio "github.com/edevhub/bazel-routines/pkg/io"
)

type DiffFileHandler struct {
	filepath string
}

func NewDiffFileHandler(filepath string) *DiffFileHandler {
	return &DiffFileHandler{filepath: filepath}
}

func (h *DiffFileHandler) TargetsList(_ context.Context) ([]string, error) {
	f, err := os.Open(h.filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file [%s]: %w", h.filepath, err)
	}

	return libio.ReadLines(f, strings.TrimSpace)
}
