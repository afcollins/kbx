package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/afcollins/kbx/internal/tui"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [file1.log file2.json ...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Interactive TUI for exploring Kubernetes audit logs and metrics.\n")
		fmt.Fprintf(os.Stderr, "Supports .log, .log.gz (audit), .json, .json.gz (metrics).\n")
		fmt.Fprintf(os.Stderr, "If no files are provided, a file picker will be shown.\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	files := flag.Args()

	initLogging()

	if err := tui.Run(files); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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
