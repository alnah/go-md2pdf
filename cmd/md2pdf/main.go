package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	md2pdf "github.com/alnah/go-md2pdf"
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
	case "completion":
		if err := runCompletion(cmdArgs, env); err != nil {
			fmt.Fprintln(env.Stderr, err)
			return 1
		}
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
	case "convert", "version", "help", "completion":
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

	// Validate worker count early
	if err := validateWorkers(flags.workers); err != nil {
		return err
	}

	// Configure GOMAXPROCS with conditional logging
	if flags.common.verbose {
		_, _ = maxprocs.Set(maxprocs.Logger(func(format string, args ...interface{}) {
			fmt.Fprintf(env.Stderr, format+"\n", args...)
		}))
	} else {
		_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...interface{}) {}))
	}

	// Load config once into env (shared across pipeline)
	if env.Config == nil {
		env.Config = config.DefaultConfig()
	}
	if flags.common.config != "" {
		env.Config, err = config.LoadConfig(flags.common.config)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}

	// Resolve asset path: CLI flag > config > embedded (default)
	assetBasePath := env.Config.Assets.BasePath
	if flags.assets.assetPath != "" {
		assetBasePath = flags.assets.assetPath
	}

	// Configure asset loader from resolved path
	if assetBasePath != "" {
		loader, err := md2pdf.NewAssetLoader(assetBasePath)
		if err != nil {
			return fmt.Errorf("initializing assets: %w", err)
		}
		env.AssetLoader = loader
		if flags.common.verbose {
			fmt.Fprintf(env.Stderr, "Using custom assets from: %s\n", assetBasePath)
		}
	}

	// Resolve template set: CLI flag > default
	templateSet, err := resolveTemplateSet(flags.assets.template, env.AssetLoader)
	if err != nil {
		return fmt.Errorf("loading template set: %w", err)
	}
	if flags.common.verbose && flags.assets.template != "" {
		fmt.Fprintf(env.Stderr, "Using template set: %s\n", templateSet.Name)
	}

	// Create pool with resolved size, asset loader, and template set
	poolSize := md2pdf.ResolvePoolSize(flags.workers)
	if flags.common.verbose {
		fmt.Fprintf(env.Stderr, "Pool size: %d\n", poolSize)
	}
	converterPool := md2pdf.NewConverterPool(poolSize,
		md2pdf.WithAssetLoader(env.AssetLoader),
		md2pdf.WithTemplateSet(templateSet),
	)
	defer converterPool.Close()

	// Wrap in adapter for local Pool interface
	pool := &poolAdapter{pool: converterPool}

	// Setup signal handling for graceful shutdown
	ctx, stop := notifyContext(context.Background())
	defer stop()

	if flags.common.verbose {
		fmt.Fprintln(env.Stderr, "Starting conversion...")
	}

	return runConvert(ctx, positionalArgs, flags, pool, env)
}

// poolAdapter adapts md2pdf.ConverterPool to the local Pool interface.
type poolAdapter struct {
	pool *md2pdf.ConverterPool
}

func (a *poolAdapter) Acquire() CLIConverter {
	return a.pool.Acquire()
}

func (a *poolAdapter) Release(c CLIConverter) {
	conv, ok := c.(*md2pdf.Converter)
	if !ok {
		panic(fmt.Sprintf("poolAdapter.Release: unexpected type %T, expected *md2pdf.Converter", c))
	}
	a.pool.Release(conv)
}

func (a *poolAdapter) Size() int {
	return a.pool.Size()
}
