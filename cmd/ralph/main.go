package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
			fmt.Println("ralph init: not yet implemented")
			return nil
		},
	}
}

func planCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Run planning loop (generates IMPLEMENTATION_PLAN.md)",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("ralph plan: not yet implemented")
			return nil
		},
	}
	cmd.Flags().IntP("max", "n", 5, "maximum iterations")
	return cmd
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run build loop (implements tasks one at a time)",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("ralph apply: not yet implemented")
			return nil
		},
	}
	cmd.Flags().IntP("max", "n", 20, "maximum iterations")
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
