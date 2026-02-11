package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/benmyles/ralph-cli/internal/config"
	"github.com/benmyles/ralph-cli/internal/docker"
	"github.com/benmyles/ralph-cli/internal/git"
	"github.com/benmyles/ralph-cli/internal/loop"
	"github.com/benmyles/ralph-cli/internal/scaffold"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "ralph",
		Short:   "Autonomous plan/build iteration using Claude Code",
		Version: version,
	}

	root.AddCommand(initCmd())
	root.AddCommand(planCmd())
	root.AddCommand(applyCmd())
	root.AddCommand(statusCmd())
	root.AddCommand(loopCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold .ralph/ in current repo",
		RunE: func(_ *cobra.Command, _ []string) error {
			repoRoot, err := git.RepoRoot()
			if err != nil {
				return fmt.Errorf("finding repo root: %w", err)
			}

			info := scaffold.Detect(repoRoot)

			if err := scaffold.RunPrompts(info, &scaffold.PromptOptions{
				In:  os.Stdin,
				Out: os.Stdout,
			}); err != nil {
				return fmt.Errorf("running prompts: %w", err)
			}

			result, err := scaffold.Generate(repoRoot, info)
			if err != nil {
				return fmt.Errorf("generating scaffold: %w", err)
			}

			scaffold.PrintSummary(os.Stdout, result)
			return nil
		},
	}
}

func planCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Run planning loop (generates .ralph/IMPLEMENTATION_PLAN.md)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			maxVal, err := cmd.Flags().GetInt("max")
			if err != nil {
				return fmt.Errorf("reading --max flag: %w", err)
			}
			branch, err := git.Branch()
			if err != nil {
				return fmt.Errorf("getting current branch: %w", err)
			}
			return docker.BuildAndRun("plan", maxVal, branch)
		},
	}
	cmd.Flags().IntP("max", "n", 0, "maximum iterations (0 = use config default)")
	return cmd
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run build loop (implements tasks one at a time)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			maxVal, err := cmd.Flags().GetInt("max")
			if err != nil {
				return fmt.Errorf("reading --max flag: %w", err)
			}
			branch, err := git.Branch()
			if err != nil {
				return fmt.Errorf("getting current branch: %w", err)
			}
			return docker.BuildAndRun("build", maxVal, branch)
		},
	}
	cmd.Flags().IntP("max", "n", 0, "maximum iterations (0 = use config default)")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Progress summary — tasks done, costs, pass/fail",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("ralph status: not yet implemented")
			return nil
		},
	}
}

// loopCmd is the hidden _loop command invoked inside Docker containers.
// Usage: ralph _loop <plan|build> [max_iterations]
func loopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_loop",
		Short:  "Internal: run iteration loop directly (used inside containers)",
		Hidden: true,
		Args:   cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			var mode loop.Mode
			switch args[0] {
			case "plan":
				mode = loop.ModePlan
			case "build":
				mode = loop.ModeBuild
			default:
				return fmt.Errorf("unknown mode: %s (expected plan or build)", args[0])
			}

			var maxIter int
			if len(args) > 1 {
				v, err := strconv.Atoi(args[1])
				if err != nil {
					return fmt.Errorf("invalid max_iterations: %w", err)
				}
				maxIter = v
			}

			return runLoop(mode, maxIter)
		},
	}
	return cmd
}

func runLoop(mode loop.Mode, maxFlag int) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	var phase config.PhaseConfig
	if mode == loop.ModePlan {
		phase = cfg.Phases.Plan
	} else {
		phase = cfg.Phases.Build
	}

	maxIterations := phase.MaxIterations
	if maxFlag > 0 {
		maxIterations = maxFlag
	}

	branch, err := git.Branch()
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	opts := &loop.Options{
		Mode:          mode,
		PromptFile:    phase.Prompt,
		MaxIterations: maxIterations,
		FreshContext:  phase.FreshContext,
		LogsDir:       "logs",
		Branch:        branch,
	}

	loopErr := loop.Run(ctx, opts, os.Stdout)
	stop()

	if ctx.Err() != nil {
		os.Exit(130)
	}

	return loopErr
}
