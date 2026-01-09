package md2pdf

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// Compile-time interface check.
var _ interface {
	Acquire() *Service
	Release(*Service)
	Size() int
	Close() error
} = (*ServicePool)(nil)

func TestResolvePoolSize(t *testing.T) {
	t.Parallel()

	gomaxprocs := runtime.GOMAXPROCS(0)

	tests := []struct {
		name    string
		workers int
		want    int
	}{
		{
			name:    "explicit takes priority",
			workers: 4,
			want:    4,
		},
		{
			name:    "explicit=1 for sequential",
			workers: 1,
			want:    1,
		},
		{
			name:    "zero uses auto calculation",
			workers: 0,
			want:    min(max(gomaxprocs/cpuDivisor, MinPoolSize), MaxPoolSize),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ResolvePoolSize(tt.workers)
			if got != tt.want {
				t.Errorf("ResolvePoolSize(%d) = %d, want %d", tt.workers, got, tt.want)
			}
		})
	}
}

func TestResolvePoolSize_Bounds(t *testing.T) {
	t.Parallel()

	t.Run("minimum is 1", func(t *testing.T) {
		t.Parallel()

		got := ResolvePoolSize(0)
		if got < MinPoolSize {
			t.Errorf("ResolvePoolSize(0) = %d, should be at least %d", got, MinPoolSize)
		}
	})

	t.Run("maximum is 8", func(t *testing.T) {
		t.Parallel()

		got := ResolvePoolSize(0)
		if got > MaxPoolSize {
			t.Errorf("ResolvePoolSize(0) = %d, should be at most %d", got, MaxPoolSize)
		}
	})

	t.Run("explicit can exceed max", func(t *testing.T) {
		t.Parallel()

		got := ResolvePoolSize(16)
		if got != 16 {
			t.Errorf("ResolvePoolSize(16) = %d, want 16", got)
		}
	})
}

func TestServicePool_AcquireRelease(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(2)
	defer pool.Close()

	// Acquire first service
	svc1 := pool.Acquire()
	if svc1 == nil {
		t.Fatal("Acquire() returned nil")
	}

	// Acquire second service
	svc2 := pool.Acquire()
	if svc2 == nil {
		t.Fatal("Acquire() returned nil")
	}

	// Services should be different instances
	if svc1 == svc2 {
		t.Error("expected different service instances")
	}

	// Release and re-acquire
	pool.Release(svc1)
	svc3 := pool.Acquire()

	if svc3 != svc1 {
		t.Error("expected to get back released service")
	}

	// Cleanup
	pool.Release(svc2)
	pool.Release(svc3)
}

func TestServicePool_Size(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
		want int
	}{
		{"size 1", 1, 1},
		{"size 4", 4, 4},
		{"size 0 becomes 1", 0, 1},
		{"negative becomes 1", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := NewServicePool(tt.size)
			defer pool.Close()

			if got := pool.Size(); got != tt.want {
				t.Errorf("Size() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestServicePool_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(4)
	defer pool.Close()

	var wg sync.WaitGroup
	iterations := 20

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := pool.Acquire()
			time.Sleep(5 * time.Millisecond) // Simulate work
			pool.Release(svc)
		}()
	}

	// Should complete without deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent access test timed out - possible deadlock")
	}
}

func TestServicePool_ClosePreventsFurtherRelease(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(2)

	svc := pool.Acquire()
	pool.Close()

	// Release after close should not panic
	pool.Release(svc) // Should be safe (no-op)
}

func TestServicePool_DoubleClose(t *testing.T) {
	t.Parallel()

	pool := NewServicePool(1)

	// First close
	if err := pool.Close(); err != nil {
		t.Errorf("first Close() error = %v", err)
	}

	// Second close should not panic
	pool.Close()
}
