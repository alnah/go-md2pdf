package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
	"go.uber.org/automaxprocs/maxprocs"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	deps := DefaultDeps()
	os.Exit(runMain(os.Args, deps))
}

// runMain is the main entry point, testable via dependency injection.
func runMain(args []string, deps *Dependencies) int {
	if len(args) < 2 {
		printUsage(deps.Stderr)
		return 1
	}

	cmd := args[1]
	cmdArgs := args[2:]

	// Legacy detection: if first arg looks like a markdown file, warn and run convert
	if !isCommand(cmd) && looksLikeMarkdown(cmd) {
		fmt.Fprintln(deps.Stderr, "DEPRECATED: use 'md2pdf convert' instead")
		cmd = "convert"
		cmdArgs = args[1:]
	}

	switch cmd {
	case "convert":
		if err := runConvertCmd(cmdArgs, deps); err != nil {
			fmt.Fprintln(deps.Stderr, err)
			return 1
		}
	case "version":
		fmt.Fprintf(deps.Stdout, "md2pdf %s\n", Version)
	case "help":
		runHelp(cmdArgs, deps)
	default:
		fmt.Fprintf(deps.Stderr, "unknown command: %s\n", cmd)
		printUsage(deps.Stderr)
		return 1
	}

	return 0
}

// isCommand checks if a string is a known command.
func isCommand(s string) bool {
	switch s {
	case "convert", "version", "help":
		return true
	}
	return false
}

// looksLikeMarkdown checks if a string looks like a markdown file.
func looksLikeMarkdown(s string) bool {
	return strings.HasSuffix(s, ".md") || strings.HasSuffix(s, ".markdown")
}

// runConvertCmd handles the convert command.
func runConvertCmd(args []string, deps *Dependencies) error {
	// Parse flags first to get workers count and verbose
	flags, positionalArgs, err := parseConvertFlags(args)
	if err != nil {
		return err
	}

	// Handle --version before any other initialization
	if flags.version {
		fmt.Fprintf(deps.Stdout, "md2pdf %s\n", Version)
		return nil
	}

	// Configure GOMAXPROCS with conditional logging
	if flags.common.verbose {
		_, _ = maxprocs.Set(maxprocs.Logger(func(format string, args ...interface{}) {
			fmt.Fprintf(deps.Stderr, format+"\n", args...)
		}))
	} else {
		_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...interface{}) {}))
	}

	// Create pool with resolved size
	poolSize := md2pdf.ResolvePoolSize(flags.workers)
	if flags.common.verbose {
		fmt.Fprintf(deps.Stderr, "Pool size: %d\n", poolSize)
	}
	servicePool := md2pdf.NewServicePool(poolSize)
	defer servicePool.Close()

	// Wrap in adapter for local Pool interface
	pool := &poolAdapter{pool: servicePool}

	// Setup signal handling for graceful shutdown
	ctx, stop := notifyContext(context.Background())
	defer stop()

	if flags.common.verbose {
		fmt.Fprintln(deps.Stderr, "Starting conversion...")
	}

	return runConvert(ctx, positionalArgs, flags, pool, deps)
}

// poolAdapter adapts md2pdf.ServicePool to the local Pool interface.
type poolAdapter struct {
	pool *md2pdf.ServicePool
}

func (a *poolAdapter) Acquire() Converter {
	return a.pool.Acquire()
}

func (a *poolAdapter) Release(c Converter) {
	svc, ok := c.(*md2pdf.Service)
	if !ok {
		fmt.Fprintf(os.Stderr, "poolAdapter.Release: unexpected type %T, expected *md2pdf.Service\n", c)
		return
	}
	a.pool.Release(svc)
}

func (a *poolAdapter) Size() int {
	return a.pool.Size()
}
