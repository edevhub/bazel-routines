package main

import (
	"context"
	"fmt"
	"github.com/edevhub/bazel-routines/internal/routines"
	"github.com/edevhub/bazel-routines/pkg/bazel"
	loglib "github.com/edevhub/bazel-routines/pkg/log"
	"github.com/urfave/cli/v3"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	logger := loglib.DefaultJSONLogger(slog.LevelWarn)
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received signal %s, exiting...", sig)
		cancel()
	}()

	if err := command(logger).Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}

func command(_ *slog.Logger) *cli.Command {
	return &cli.Command{
		Name: "routines",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Runs loaded routines",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "tag",
						Usage: "Bazel tags to filter routines",
					},
					&cli.IntFlag{
						Name:  "concurrency",
						Usage: "Number of concurrent tasks to run",
						Value: int64(runtime.NumCPU()),
					},
					&cli.StringFlag{
						Name:  "diff",
						Usage: "Diff file to limit scope of the routines by changed targets only",
						Action: func(ctx context.Context, c *cli.Command, s string) error {
							if s == "" {
								return nil
							}
							if _, err := os.Stat(s); err != nil {
								return fmt.Errorf("failed to open file (%s): %w", s, err)
							}
							return nil
						},
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					var cfg bazel.ClientConfig
					if err := cfg.Load(); err != nil {
						return fmt.Errorf("failed to load bazel config: %w", err)
					}
					client := bazel.NewClient(&cfg)
					registryBuilder := routines.NewRegistryBuilder(client)
					reg, err := registryBuilder.Load(ctx)
					if err != nil {
						return fmt.Errorf("failed to load registry: %w", err)
					}

					for label, routine := range reg {
						fmt.Printf("Routine: %s\n", label)
						for _, task := range routine.Tasks {
							fmt.Printf(" - %s\n", task.Label)
						}
					}
					return nil
				},
			},
		},
	}
}
