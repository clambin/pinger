package main

import (
	"log/slog"
	"os"

	"github.com/clambin/pinger/internal/cmd"
)

var version = "change-me"

func main() {
	cmd.Cmd.Version = version
	if err := cmd.Cmd.Execute(); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}
}
