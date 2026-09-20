package main

import (
	"fmt"
	"log/slog"
	"os"
)

type dependencies struct {
}

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	deps, err := makeDependencies()
	if err != nil {
		return fmt.Errorf("make dependencies: %w", err)
	}

	r := newRouter(deps)

	if err := r.Run(":8088"); err != nil {
		return fmt.Errorf("failed to run server: %v", err)
	}
	return nil
}

func makeDependencies() (*dependencies, error) {
	return &dependencies{}, nil
}
