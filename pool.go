package md2pdf

import (
	"errors"
	"runtime"
	"sync"
)

// Pool sizing constants.
const (
	// MinPoolSize ensures at least one worker is available.
	MinPoolSize = 1

	// MaxPoolSize caps browser instances to limit memory (~200MB each).
	MaxPoolSize = 8

	// cpuDivisor leaves headroom for Chrome child processes.
	cpuDivisor = 2
)

// ServicePool manages a pool of Service instances for parallel processing.
// Each service has its own browser instance, enabling true parallelism.
// Services are created lazily on first acquire to avoid startup delay.
type ServicePool struct {
	size     int
	opts     []Option
	services []*Service
	sem      chan *Service
	mu       sync.Mutex
	created  int
	closed   bool
	initErr  error // First error encountered during service creation
}

// NewServicePool creates a pool with capacity for n Service instances.
// Services are created lazily when acquired, not at pool creation.
// Options are applied to each service when created.
func NewServicePool(n int, opts ...Option) *ServicePool {
	if n < 1 {
		n = 1
	}

	return &ServicePool{
		size:     n,
		opts:     opts,
		services: make([]*Service, 0, n),
		sem:      make(chan *Service, n),
	}
}

// Acquire gets a service from the pool, creating one if needed.
// Blocks if all services are in use.
// Returns nil and sets internal error if service creation fails.
// Use InitError() to check for initialization failures.
func (p *ServicePool) Acquire() *Service {
	// Try to get an existing service (non-blocking)
	select {
	case svc := <-p.sem:
		return svc
	default:
	}

	// Check if we can create a new service
	p.mu.Lock()
	if p.initErr != nil {
		p.mu.Unlock()
		return nil
	}
	if p.created < p.size {
		p.created++
		p.mu.Unlock()

		// Create new service outside the lock
		svc, err := New(p.opts...)
		if err != nil {
			p.mu.Lock()
			if p.initErr == nil {
				p.initErr = err
			}
			p.created--
			p.mu.Unlock()
			return nil
		}

		p.mu.Lock()
		p.services = append(p.services, svc)
		p.mu.Unlock()

		return svc
	}
	p.mu.Unlock()

	// All services created, wait for one to be released
	return <-p.sem
}

// InitError returns the first error encountered during service creation.
// Returns nil if all services were created successfully.
func (p *ServicePool) InitError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.initErr
}

// Release returns a service to the pool.
// The lock is released before sending to avoid deadlock when channel is full.
func (p *ServicePool) Release(svc *Service) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.sem <- svc
}

// Close releases all browser resources.
// Returns an aggregated error if multiple services fail to close.
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

	var errs []error
	for _, svc := range services {
		if err := svc.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Size returns the pool capacity.
func (p *ServicePool) Size() int {
	return p.size
}

// ResolvePoolSize determines the optimal pool size.
// Priority: explicit workers > GOMAXPROCS-based calculation.
// Exported for use by servers and CLIs.
func ResolvePoolSize(workers int) int {
	// Explicit value takes priority
	if workers > 0 {
		return workers
	}

	// Auto-calculate based on GOMAXPROCS (adjusted by automaxprocs for containers)
	available := runtime.GOMAXPROCS(0)
	n := available / cpuDivisor

	if n < MinPoolSize {
		return MinPoolSize
	}
	if n > MaxPoolSize {
		return MaxPoolSize
	}
	return n
}
