package main

import (
	"fmt"
	"os"

	md2pdf "github.com/alnah/go-md2pdf"
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
	poolSize := md2pdf.ResolvePoolSize(flags.workers)
	if flags.verbose {
		fmt.Fprintf(os.Stderr, "Pool size: %d\n", poolSize)
	}
	servicePool := md2pdf.NewServicePool(poolSize)
	defer servicePool.Close()

	// Wrap in adapter for local Pool interface (used for testing)
	pool := &poolAdapter{pool: servicePool}

	if flags.verbose {
		fmt.Fprintln(os.Stderr, "Starting conversion...")
	}

	if err := run(os.Args, pool); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
