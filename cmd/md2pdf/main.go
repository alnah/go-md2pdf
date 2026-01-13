package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/assets"
	"github.com/alnah/go-md2pdf/internal/config"
	"go.uber.org/automaxprocs/maxprocs"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	env := DefaultEnv()
	os.Exit(runMain(os.Args, env))
}

// runMain is the main entry point, testable via dependency injection.
func runMain(args []string, env *Environment) int {
	if len(args) < 2 {
		printUsage(env.Stderr)
		return 1
	}

	cmd := args[1]
	cmdArgs := args[2:]

	// Legacy detection: if first arg looks like a markdown file, warn and run convert
	if !isCommand(cmd) && looksLikeMarkdown(cmd) {
		fmt.Fprintln(env.Stderr, "DEPRECATED: use 'md2pdf convert' instead")
		cmd = "convert"
		cmdArgs = args[1:]
	}

	switch cmd {
	case "convert":
		if err := runConvertCmd(cmdArgs, env); err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
	case "version":
		fmt.Fprintf(env.Stdout, "md2pdf %s\n", Version)
	case "help":
		runHelp(cmdArgs, env)
	default:
		fmt.Fprintf(env.Stderr, "unknown command: %s\n", cmd)
		printUsage(env.Stderr)
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
func runConvertCmd(args []string, env *Environment) error {
	// Parse flags first to get workers count and verbose
	flags, positionalArgs, err := parseConvertFlags(args)
	if err != nil {
		return err
	}

	// Handle --version before any other initialization
	if flags.version {
		fmt.Fprintf(env.Stdout, "md2pdf %s\n", Version)
		return nil
	}

	// Configure GOMAXPROCS with conditional logging
	if flags.common.verbose {
		_, _ = maxprocs.Set(maxprocs.Logger(func(format string, args ...interface{}) {
			fmt.Fprintf(env.Stderr, format+"\n", args...)
		}))
	} else {
		_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...interface{}) {}))
	}

	// Load config early to get assets.basePath for the pool
	cfg := config.DefaultConfig()
	if flags.common.config != "" {
		cfg, err = config.LoadConfig(flags.common.config)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}

	// Configure asset loader from config (custom path overrides default)
	if cfg.Assets.BasePath != "" {
		resolver, err := assets.NewAssetResolver(cfg.Assets.BasePath)
		if err != nil {
			return fmt.Errorf("initializing assets: %w", err)
		}
		env.AssetLoader = resolver
		if flags.common.verbose {
			fmt.Fprintf(env.Stderr, "Using custom assets from: %s\n", cfg.Assets.BasePath)
		}
	}

	// Create pool with resolved size and asset loader
	poolSize := md2pdf.ResolvePoolSize(flags.workers)
	if flags.common.verbose {
		fmt.Fprintf(env.Stderr, "Pool size: %d\n", poolSize)
	}
	servicePool := md2pdf.NewServicePool(poolSize, md2pdf.WithAssetLoader(env.AssetLoader))
	defer servicePool.Close()

	// Wrap in adapter for local Pool interface
	pool := &poolAdapter{pool: servicePool}

	// Setup signal handling for graceful shutdown
	ctx, stop := notifyContext(context.Background())
	defer stop()

	if flags.common.verbose {
		fmt.Fprintln(env.Stderr, "Starting conversion...")
	}

	return runConvert(ctx, positionalArgs, flags, pool, env)
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
		panic(fmt.Sprintf("poolAdapter.Release: unexpected type %T, expected *md2pdf.Service", c))
	}
	a.pool.Release(svc)
}

func (a *poolAdapter) Size() int {
	return a.pool.Size()
}
