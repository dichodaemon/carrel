package main

import (
	"fmt"
	"os"

	"github.com/dichodaemon/carrel/internal/registry"
)

func mustOpenRegistry() registry.Registry {
	if err := os.MkdirAll("/workspace/.carrel", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "registry dir: %v\n", err)
		os.Exit(1)
	}
	reg, err := registry.New("/workspace/.carrel")
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(1)
	}
	return reg
}
