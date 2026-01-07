package main

import (
	"runtime"
	"sync"

	md2pdf "github.com/alnah/go-md2pdf"
)

// ServicePool manages a pool of md2pdf.Service instances for parallel processing.
// Each service has its own browser instance, enabling true parallelism.
// Services are created lazily on first acquire to avoid startup delay.
type ServicePool struct {
	size     int
	services []*md2pdf.Service
	sem      chan Converter
	mu       sync.Mutex
	created  int
	closed   bool
}

// NewServicePool creates a pool with capacity for n Service instances.
// Services are created lazily when acquired, not at pool creation.
func NewServicePool(n int) *ServicePool {
	if n < 1 {
		n = 1
	}

	return &ServicePool{
		size:     n,
		services: make([]*md2pdf.Service, 0, n),
		sem:      make(chan Converter, n),
	}
}

// Compile-time check that ServicePool implements Pool.
var _ Pool = (*ServicePool)(nil)

// Acquire gets a service from the pool, creating one if needed.
// Blocks if all services are in use.
func (p *ServicePool) Acquire() Converter {
	// Try to get an existing service (non-blocking)
	select {
	case svc := <-p.sem:
		return svc
	default:
	}

	// Check if we can create a new service
	p.mu.Lock()
	if p.created < p.size {
		p.created++
		p.mu.Unlock()

		// Create new service outside the lock
		svc := md2pdf.New()

		p.mu.Lock()
		p.services = append(p.services, svc)
		p.mu.Unlock()

		return svc
	}
	p.mu.Unlock()

	// All services created, wait for one to be released
	return <-p.sem
}

// Release returns a service to the pool.
func (p *ServicePool) Release(svc Converter) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		p.sem <- svc
	}
}

// Close releases all browser resources.
func (p *ServicePool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	close(p.sem)
	services := p.services
	p.mu.Unlock()

	var lastErr error
	for _, svc := range services {
		if err := svc.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Size returns the pool capacity.
func (p *ServicePool) Size() int {
	return p.size
}

// resolvePoolSize determines the optimal pool size.
// Priority: explicit flag > GOMAXPROCS-based calculation.
func resolvePoolSize(flagWorkers int) int {
	// Explicit flag takes priority
	if flagWorkers > 0 {
		return flagWorkers
	}

	// Auto-calculate based on GOMAXPROCS (adjusted by automaxprocs for containers)
	available := runtime.GOMAXPROCS(0)
	n := available / 2

	// Minimum 1, maximum 8
	if n < 1 {
		return 1
	}
	if n > 8 {
		return 8
	}
	return n
}
