package main

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestResolvePoolSize(t *testing.T) {
	gomaxprocs := runtime.GOMAXPROCS(0)

	tests := []struct {
		name        string
		flagWorkers int
		want        int
	}{
		{
			name:        "flag takes priority",
			flagWorkers: 4,
			want:        4,
		},
		{
			name:        "flag=1 for sequential",
			flagWorkers: 1,
			want:        1,
		},
		{
			name:        "flag=0 uses auto calculation",
			flagWorkers: 0,
			want:        min(max(gomaxprocs/2, 1), 8),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePoolSize(tt.flagWorkers)
			if got != tt.want {
				t.Errorf("resolvePoolSize(%d) = %d, want %d", tt.flagWorkers, got, tt.want)
			}
		})
	}
}

func TestResolvePoolSize_Bounds(t *testing.T) {
	// Test minimum bound
	t.Run("minimum is 1", func(t *testing.T) {
		got := resolvePoolSize(0)
		if got < 1 {
			t.Errorf("resolvePoolSize(0) = %d, should be at least 1", got)
		}
	})

	// Test maximum bound
	t.Run("maximum is 8", func(t *testing.T) {
		got := resolvePoolSize(0)
		if got > 8 {
			t.Errorf("resolvePoolSize(0) = %d, should be at most 8", got)
		}
	})

	// Explicit flag can exceed 8
	t.Run("explicit flag can exceed max", func(t *testing.T) {
		got := resolvePoolSize(16)
		if got != 16 {
			t.Errorf("resolvePoolSize(16) = %d, want 16", got)
		}
	})
}

func TestServicePool_AcquireRelease(t *testing.T) {
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
			pool := NewServicePool(tt.size)
			defer pool.Close()

			if got := pool.Size(); got != tt.want {
				t.Errorf("Size() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestServicePool_ConcurrentAccess(t *testing.T) {
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
	pool := NewServicePool(2)

	svc := pool.Acquire()
	pool.Close()

	// Release after close should not panic
	pool.Release(svc) // Should be safe (no-op)
}

func TestServicePool_DoubleClose(t *testing.T) {
	pool := NewServicePool(1)

	// First close
	if err := pool.Close(); err != nil {
		t.Errorf("first Close() error = %v", err)
	}

	// Second close should not panic (but may error)
	// We just verify it doesn't panic
	pool.Close()
}
