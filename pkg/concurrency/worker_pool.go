package concurrency

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
)

var (
	ErrPoolFull    = errors.New("worker pool is full")
	ErrPoolClosed  = errors.New("worker pool is closed")
	ErrTaskTimeout = errors.New("task execution timeout")
)

// Task represents a unit of work to be executed
type Task interface {
	Execute(ctx context.Context) error
	Name() string
}

// TaskFunc allows using functions as tasks
type TaskFunc struct {
	name string
	fn   func(ctx context.Context) error
}

func NewTaskFunc(name string, fn func(ctx context.Context) error) Task {
	return &TaskFunc{name: name, fn: fn}
}

func (t *TaskFunc) Execute(ctx context.Context) error {
	return t.fn(ctx)
}

func (t *TaskFunc) Name() string {
	return t.name
}

// PoolMetrics tracks worker pool statistics
type PoolMetrics struct {
	ActiveWorkers   int32
	QueuedTasks     int32
	CompletedTasks  uint64
	FailedTasks     uint64
	TotalDuration   int64 // nanoseconds
	MaxDuration     int64 // nanoseconds
	LastTaskTime    time.Time
}

// WorkerPool manages concurrent task execution
type WorkerPool struct {
	maxWorkers  int
	queue       chan Task
	sem         chan struct{}
	metrics     *PoolMetrics
	logger      logging.Logger
	wg          sync.WaitGroup
	mu          sync.RWMutex
	closed      bool
	taskTimeout time.Duration
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers, queueSize int, taskTimeout time.Duration, logger logging.Logger) *WorkerPool {
	return &WorkerPool{
		maxWorkers:  maxWorkers,
		queue:       make(chan Task, queueSize),
		sem:         make(chan struct{}, maxWorkers),
		metrics:     &PoolMetrics{},
		logger:      logger.WithComponent("worker_pool"),
		taskTimeout: taskTimeout,
	}
}

// Submit adds a task to the worker pool
func (wp *WorkerPool) Submit(ctx context.Context, task Task) error {
	wp.mu.RLock()
	if wp.closed {
		wp.mu.RUnlock()
		return ErrPoolClosed
	}
	wp.mu.RUnlock()

	// Try to acquire semaphore (non-blocking)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case wp.sem <- struct{}{}:
		// Successfully acquired semaphore
		atomic.AddInt32(&wp.metrics.ActiveWorkers, 1)
		wp.wg.Add(1)
		
		go wp.runWorker(ctx, task)
		return nil
	default:
		// Pool is full, try to queue
		select {
		case wp.queue <- task:
			atomic.AddInt32(&wp.metrics.QueuedTasks, 1)
			wp.logger.Debug("task_queued",
				"task", task.Name(),
				"queue_size", atomic.LoadInt32(&wp.metrics.QueuedTasks))
			return nil
		default:
			wp.logger.Warn("pool_full",
				"task", task.Name(),
				"active_workers", atomic.LoadInt32(&wp.metrics.ActiveWorkers),
				"queued_tasks", atomic.LoadInt32(&wp.metrics.QueuedTasks))
			return ErrPoolFull
		}
	}
}

// runWorker executes a task and continues processing from queue
func (wp *WorkerPool) runWorker(ctx context.Context, initialTask Task) {
	defer func() {
		<-wp.sem // Release semaphore
		atomic.AddInt32(&wp.metrics.ActiveWorkers, -1)
		wp.wg.Done()
		
		if r := recover(); r != nil {
			wp.logger.Error("worker_panic",
				"panic", r,
				"stack", string(debug.Stack()))
		}
	}()

	// Execute initial task
	wp.executeTask(ctx, initialTask)

	// Process queued tasks
	for {
		select {
		case task := <-wp.queue:
			atomic.AddInt32(&wp.metrics.QueuedTasks, -1)
			wp.executeTask(ctx, task)
		default:
			// No more queued tasks
			return
		}
	}
}

// executeTask runs a single task with timeout and monitoring
func (wp *WorkerPool) executeTask(ctx context.Context, task Task) {
	start := time.Now()
	taskCtx, cancel := context.WithTimeout(ctx, wp.taskTimeout)
	defer cancel()

	// Create task-specific logger
	taskLogger := wp.logger.WithComponent(fmt.Sprintf("task.%s", task.Name()))
	
	taskLogger.Debug("task_started",
		"active_workers", atomic.LoadInt32(&wp.metrics.ActiveWorkers))

	// Execute task
	err := task.Execute(taskCtx)
	
	duration := time.Since(start)
	atomic.AddInt64(&wp.metrics.TotalDuration, duration.Nanoseconds())
	
	// Update max duration
	for {
		current := atomic.LoadInt64(&wp.metrics.MaxDuration)
		if duration.Nanoseconds() <= current || atomic.CompareAndSwapInt64(&wp.metrics.MaxDuration, current, duration.Nanoseconds()) {
			break
		}
	}

	if err != nil {
		atomic.AddUint64(&wp.metrics.FailedTasks, 1)
		taskLogger.Error("task_failed",
			"error", err,
			"duration_ms", duration.Milliseconds())
	} else {
		atomic.AddUint64(&wp.metrics.CompletedTasks, 1)
		taskLogger.Debug("task_completed",
			"duration_ms", duration.Milliseconds())
	}

	wp.metrics.LastTaskTime = time.Now()
}

// GetMetrics returns current pool metrics
func (wp *WorkerPool) GetMetrics() PoolMetrics {
	completed := atomic.LoadUint64(&wp.metrics.CompletedTasks)
	totalDuration := atomic.LoadInt64(&wp.metrics.TotalDuration)
	
	avgDuration := int64(0)
	if completed > 0 {
		avgDuration = totalDuration / int64(completed)
	}

	return PoolMetrics{
		ActiveWorkers:  atomic.LoadInt32(&wp.metrics.ActiveWorkers),
		QueuedTasks:    atomic.LoadInt32(&wp.metrics.QueuedTasks),
		CompletedTasks: completed,
		FailedTasks:    atomic.LoadUint64(&wp.metrics.FailedTasks),
		TotalDuration:  avgDuration,
		MaxDuration:    atomic.LoadInt64(&wp.metrics.MaxDuration),
		LastTaskTime:   wp.metrics.LastTaskTime,
	}
}

// Close shuts down the worker pool
func (wp *WorkerPool) Close(ctx context.Context) error {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return nil
	}
	wp.closed = true
	wp.mu.Unlock()

	wp.logger.Info("pool_closing",
		"active_workers", atomic.LoadInt32(&wp.metrics.ActiveWorkers),
		"queued_tasks", atomic.LoadInt32(&wp.metrics.QueuedTasks))

	// Close queue to prevent new submissions
	close(wp.queue)

	// Wait for workers to complete or timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.logger.Info("pool_closed", "clean_shutdown", true)
		return nil
	case <-ctx.Done():
		wp.logger.Warn("pool_close_timeout",
			"remaining_workers", atomic.LoadInt32(&wp.metrics.ActiveWorkers))
		return ctx.Err()
	}
}

// LogMetrics logs current pool metrics (for LLM troubleshooting)
func (wp *WorkerPool) LogMetrics() {
	metrics := wp.GetMetrics()
	wp.logger.Info("pool_metrics",
		"active_workers", metrics.ActiveWorkers,
		"queued_tasks", metrics.QueuedTasks,
		"completed_tasks", metrics.CompletedTasks,
		"failed_tasks", metrics.FailedTasks,
		"avg_duration_ms", time.Duration(metrics.TotalDuration).Milliseconds(),
		"max_duration_ms", time.Duration(metrics.MaxDuration).Milliseconds(),
		"last_task_time", metrics.LastTaskTime.Format(time.RFC3339))
}