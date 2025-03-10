package routines

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/edevhub/bazel-routines/internal/task"
	"github.com/edevhub/bazel-routines/pkg/bazel"
)

// ManifestRuleKind is routine manifest rule kind
const ManifestRuleKind = "routine_manifest"

type Registry map[task.Label]*task.Routine

type RegistryBuilder struct {
	c *bazel.Client
}

func NewRegistryBuilder(c *bazel.Client) *RegistryBuilder {
	return &RegistryBuilder{c: c}
}

func (b *RegistryBuilder) Load(ctx context.Context) (Registry, error) {
	result := struct {
		Query xml.Name   `xml:"query"`
		Rules []*XMLRule `xml:"rule"`
	}{}

	out := bazel.XmlOutput{
		Result: &result,
	}
	r := b.c.Query(ctx, fmt.Sprintf("kind(%s, //...)", ManifestRuleKind), &out)
	if r.Err != nil {
		return nil, fmt.Errorf("failed to load manifests: %w", r.Err)
	}

	candidates, err := DecodeCandidatesXML(result.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to decode loaded manifests: %w", err)
	}

	reg := make(Registry)
	for _, c := range candidates {
		label := task.Label(c.Label)
		tasks := make([]*task.Task, len(c.Targets))
		for i, t := range c.Targets {
			p, err := b.c.Run(ctx, t)
			if err != nil {
				return nil, fmt.Errorf("failed to prepare run task[%s]: %w", t, err)
			}
			tasks[i] = task.NewTask(task.Label(t), p)
		}
		reg[label] = &task.Routine{
			Label:   label,
			Summary: c.Summary,
			Tasks:   tasks,
		}
	}
	return reg, nil
}

func (r Registry) Filter(labels ...task.Label) Registry {
	if len(labels) == 0 {
		return r
	}
	reg := make(Registry)
	for _, l := range labels {
		if v, ok := r[l]; ok {
			reg[l] = v
		}
	}
	return reg
}
