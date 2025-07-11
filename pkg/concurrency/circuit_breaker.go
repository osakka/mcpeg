package concurrency

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/yourusername/mcpeg/pkg/logging"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	maxFailures     int
	resetTimeout    time.Duration
	halfOpenMax     int
	successThreshold int
	
	mu              sync.RWMutex
	state           State
	failures        int
	successes       int
	halfOpenCount   int
	lastFailureTime time.Time
	lastStateChange time.Time
	generation      uint64
	
	logger          logging.Logger
}

// CircuitBreakerConfig contains configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name             string
	MaxFailures      int
	ResetTimeout     time.Duration
	HalfOpenMax      int
	SuccessThreshold int
	Logger           logging.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = config.HalfOpenMax
	}
	
	cb := &CircuitBreaker{
		name:             config.Name,
		maxFailures:      config.MaxFailures,
		resetTimeout:     config.ResetTimeout,
		halfOpenMax:      config.HalfOpenMax,
		successThreshold: config.SuccessThreshold,
		state:            StateClosed,
		lastStateChange:  time.Now(),
		logger:           config.Logger.WithComponent("circuit_breaker." + config.Name),
	}
	
	cb.logger.Info("circuit_breaker_created",
		"max_failures", cb.maxFailures,
		"reset_timeout", cb.resetTimeout,
		"half_open_max", cb.halfOpenMax)
	
	return cb
}

// Execute runs the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.beforeExecute(); err != nil {
		return err
	}
	
	// Execute the function
	err := fn()
	
	cb.afterExecute(err)
	return err
}

// beforeExecute checks if execution is allowed
func (cb *CircuitBreaker) beforeExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	now := time.Now()
	
	switch cb.state {
	case StateClosed:
		// Normal operation
		return nil
		
	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenCount = 1
			return nil
		}
		
		cb.logger.Debug("circuit_breaker_rejected",
			"state", "open",
			"time_until_reset", cb.resetTimeout-now.Sub(cb.lastFailureTime))
		return ErrCircuitOpen
		
	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenCount >= cb.halfOpenMax {
			cb.logger.Debug("circuit_breaker_rejected",
				"state", "half-open",
				"reason", "max_requests_reached",
				"count", cb.halfOpenCount)
			return ErrTooManyRequests
		}
		cb.halfOpenCount++
		return nil
		
	default:
		return errors.New("unknown circuit breaker state")
	}
}

// afterExecute updates the circuit breaker state based on the result
func (cb *CircuitBreaker) afterExecute(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

// recordFailure handles a failed execution
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.transitionTo(StateOpen)
			cb.logger.Error("circuit_breaker_opened",
				"failures", cb.failures,
				"threshold", cb.maxFailures,
				"reset_timeout", cb.resetTimeout,
				"suggested_actions", []string{
					"Check backend service health",
					"Review error logs for root cause",
					"Consider increasing timeout values",
					"Verify network connectivity",
				})
		}
		
	case StateHalfOpen:
		// Single failure in half-open state reopens the circuit
		cb.transitionTo(StateOpen)
		cb.logger.Warn("circuit_breaker_reopened",
			"reason", "failure_in_half_open_state",
			"reset_timeout", cb.resetTimeout)
	}
}

// recordSuccess handles a successful execution
func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case StateClosed:
		// Reset failure count on success in closed state
		cb.failures = 0
		
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.successThreshold {
			// Enough successes, close the circuit
			cb.transitionTo(StateClosed)
			cb.logger.Info("circuit_breaker_closed",
				"successes", cb.successes,
				"threshold", cb.successThreshold,
				"recovery_time", time.Since(cb.lastFailureTime))
		}
	}
}

// transitionTo changes the circuit breaker state
func (cb *CircuitBreaker) transitionTo(newState State) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()
	cb.generation++
	
	// Reset counters based on new state
	switch newState {
	case StateClosed:
		cb.failures = 0
		cb.successes = 0
		cb.halfOpenCount = 0
	case StateHalfOpen:
		cb.successes = 0
		cb.halfOpenCount = 0
	}
	
	cb.logger.Info("circuit_breaker_state_change",
		"from", oldState.String(),
		"to", newState.String(),
		"generation", cb.generation)
}

// GetState returns the current state and metadata
func (cb *CircuitBreaker) GetState() (State, CircuitBreakerStatus) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	status := CircuitBreakerStatus{
		State:           cb.state.String(),
		Failures:        cb.failures,
		Successes:       cb.successes,
		HalfOpenCount:   cb.halfOpenCount,
		LastFailureTime: cb.lastFailureTime,
		LastStateChange: cb.lastStateChange,
		Generation:      cb.generation,
	}
	
	if cb.state == StateOpen {
		status.TimeUntilReset = cb.resetTimeout - time.Since(cb.lastFailureTime)
		if status.TimeUntilReset < 0 {
			status.TimeUntilReset = 0
		}
	}
	
	return cb.state, status
}

// CircuitBreakerStatus contains detailed circuit breaker status
type CircuitBreakerStatus struct {
	State           string        `json:"state"`
	Failures        int           `json:"failures"`
	Successes       int           `json:"successes"`
	HalfOpenCount   int           `json:"half_open_count"`
	LastFailureTime time.Time     `json:"last_failure_time"`
	LastStateChange time.Time     `json:"last_state_change"`
	Generation      uint64        `json:"generation"`
	TimeUntilReset  time.Duration `json:"time_until_reset,omitempty"`
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.logger.Info("circuit_breaker_manual_reset",
		"previous_state", cb.state.String())
	
	cb.transitionTo(StateClosed)
}

// LogStatus logs the current circuit breaker status for debugging
func (cb *CircuitBreaker) LogStatus() {
	_, status := cb.GetState()
	
	cb.logger.Info("circuit_breaker_status",
		"state", status.State,
		"failures", status.Failures,
		"successes", status.Successes,
		"half_open_count", status.HalfOpenCount,
		"last_failure", status.LastFailureTime.Format(time.RFC3339),
		"last_change", status.LastStateChange.Format(time.RFC3339),
		"generation", status.Generation,
		"time_until_reset_seconds", status.TimeUntilReset.Seconds())
}

// CircuitBreakerGroup manages multiple circuit breakers
type CircuitBreakerGroup struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	logger   logging.Logger
}

// NewCircuitBreakerGroup creates a new circuit breaker group
func NewCircuitBreakerGroup(logger logging.Logger) *CircuitBreakerGroup {
	return &CircuitBreakerGroup{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger.WithComponent("circuit_breaker_group"),
	}
}

// Add adds a circuit breaker to the group
func (g *CircuitBreakerGroup) Add(name string, cb *CircuitBreaker) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.breakers[name] = cb
	g.logger.Debug("circuit_breaker_added", "name", name)
}

// Get retrieves a circuit breaker by name
func (g *CircuitBreakerGroup) Get(name string) (*CircuitBreaker, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	cb, ok := g.breakers[name]
	return cb, ok
}

// GetStatus returns the status of all circuit breakers
func (g *CircuitBreakerGroup) GetStatus() map[string]CircuitBreakerStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	status := make(map[string]CircuitBreakerStatus)
	for name, cb := range g.breakers {
		_, cbStatus := cb.GetState()
		status[name] = cbStatus
	}
	
	return status
}

// LogAllStatus logs the status of all circuit breakers
func (g *CircuitBreakerGroup) LogAllStatus() {
	status := g.GetStatus()
	
	g.logger.Info("circuit_breaker_group_status",
		"total_breakers", len(status),
		"breakers", status)
}