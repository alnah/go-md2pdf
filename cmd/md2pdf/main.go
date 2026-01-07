package main

import (
	"fmt"
	"os"

	"go.uber.org/automaxprocs/maxprocs"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	// Parse flags first to get workers count and verbose
	flags, _, err := parseFlags(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Configure GOMAXPROCS with conditional logging
	// Error ignored: maxprocs.Set only fails if GOMAXPROCS env is invalid,
	// in which case Go runtime defaults apply and the program continues safely.
	if flags.verbose {
		_, _ = maxprocs.Set(maxprocs.Logger(func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}))
	} else {
		_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...interface{}) {}))
	}

	// Create pool with resolved size
	poolSize := resolvePoolSize(flags.workers)
	if flags.verbose {
		fmt.Fprintf(os.Stderr, "Pool size: %d\n", poolSize)
	}
	pool := NewServicePool(poolSize)
	defer pool.Close()

	if flags.verbose {
		fmt.Fprintln(os.Stderr, "Starting conversion...")
	}

	if err := run(os.Args, pool); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
