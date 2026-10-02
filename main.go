package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/afcollins/kbx/internal/tui"
	"github.com/spf13/cobra"
)

var Version = "dev"
var BuildDate = ""
var GitCommit = ""

func main() {
	rootCmd := newRootCmd(func(files []string) error {
		initLogging()
		return tui.Run(files)
	})
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd(run func([]string) error) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kbx [file1.log file2.json ...]",
		Short: "Explore Kubernetes audit logs and kube-burner metrics",
		Long: "Interactive TUI for exploring Kubernetes audit logs and kube-burner metrics.\n" +
			"Supports .log, .log.gz (audit), .json, and .json.gz (metrics).\n" +
			"If no files are provided, a file picker is shown.",
		Args: cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return run(args)
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the build revision",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Version:", Version)
			fmt.Fprintln(cmd.OutOrStdout(), "Git Commit:", GitCommit)
			fmt.Fprintln(cmd.OutOrStdout(), "Build Date:", BuildDate)
		},
	}
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

func initLogging() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".kbx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Warn("unable to create log dir", "error", err)
	}
	logPath := filepath.Join(dir, "kbx.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	slog.Info("kbx started", "args", os.Args[1:])
}
