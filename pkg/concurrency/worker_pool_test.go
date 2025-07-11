package concurrency

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
)

func TestWorkerPool(t *testing.T) {
	logger := logging.New("test")
	
	t.Run("executes tasks successfully", func(t *testing.T) {
		pool := NewWorkerPool(5, 10, 1*time.Second, logger)
		defer pool.Close(context.Background())
		
		var counter int32
		task := NewTaskFunc("increment", func(ctx context.Context) error {
			atomic.AddInt32(&counter, 1)
			return nil
		})
		
		// Submit 10 tasks
		for i := 0; i < 10; i++ {
			err := pool.Submit(context.Background(), task)
			if err != nil {
				t.Fatalf("failed to submit task: %v", err)
			}
		}
		
		// Wait for completion
		time.Sleep(100 * time.Millisecond)
		
		if atomic.LoadInt32(&counter) != 10 {
			t.Errorf("expected counter=10, got %d", counter)
		}
		
		metrics := pool.GetMetrics()
		if metrics.CompletedTasks != 10 {
			t.Errorf("expected 10 completed tasks, got %d", metrics.CompletedTasks)
		}
	})
	
	t.Run("handles task errors", func(t *testing.T) {
		pool := NewWorkerPool(2, 5, 1*time.Second, logger)
		defer pool.Close(context.Background())
		
		task := NewTaskFunc("error_task", func(ctx context.Context) error {
			return errors.New("task error")
		})
		
		err := pool.Submit(context.Background(), task)
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
		
		time.Sleep(50 * time.Millisecond)
		
		metrics := pool.GetMetrics()
		if metrics.FailedTasks != 1 {
			t.Errorf("expected 1 failed task, got %d", metrics.FailedTasks)
		}
	})
	
	t.Run("respects pool size limit", func(t *testing.T) {
		pool := NewWorkerPool(1, 0, 1*time.Second, logger)
		defer pool.Close(context.Background())
		
		// First task should succeed
		slowTask := NewTaskFunc("slow", func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		
		err := pool.Submit(context.Background(), slowTask)
		if err != nil {
			t.Fatalf("first task should succeed: %v", err)
		}
		
		// Second task should fail (pool full, no queue)
		err = pool.Submit(context.Background(), slowTask)
		if err != ErrPoolFull {
			t.Errorf("expected ErrPoolFull, got %v", err)
		}
	})
	
	t.Run("queues tasks when pool is full", func(t *testing.T) {
		pool := NewWorkerPool(1, 5, 1*time.Second, logger)
		defer pool.Close(context.Background())
		
		var completed int32
		task := NewTaskFunc("queued", func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		})
		
		// Submit 5 tasks (1 running, 4 queued)
		for i := 0; i < 5; i++ {
			err := pool.Submit(context.Background(), task)
			if err != nil {
				t.Fatalf("failed to submit task %d: %v", i, err)
			}
		}
		
		// Wait for all to complete
		time.Sleep(100 * time.Millisecond)
		
		if atomic.LoadInt32(&completed) != 5 {
			t.Errorf("expected 5 completed tasks, got %d", completed)
		}
	})
	
	t.Run("graceful shutdown", func(t *testing.T) {
		pool := NewWorkerPool(2, 10, 1*time.Second, logger)
		
		var started, completed int32
		task := NewTaskFunc("shutdown_test", func(ctx context.Context) error {
			atomic.AddInt32(&started, 1)
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		})
		
		// Submit tasks
		for i := 0; i < 5; i++ {
			pool.Submit(context.Background(), task)
		}
		
		// Close with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		
		err := pool.Close(ctx)
		if err != nil {
			t.Errorf("close failed: %v", err)
		}
		
		// All started tasks should complete
		if started := atomic.LoadInt32(&started); started > 0 {
			if completed := atomic.LoadInt32(&completed); completed != started {
				t.Errorf("started %d tasks but only %d completed", started, completed)
			}
		}
	})
}