package registry

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
)

// LoadBalancer handles service selection and load balancing
type LoadBalancer struct {
	registry *ServiceRegistry
	logger   logging.Logger
	metrics  metrics.Metrics
	config   LoadBalancerConfig
	
	// Per-service state tracking
	serviceState map[string]*ServiceState
	mutex        sync.RWMutex
}

// LoadBalancerConfig configures load balancing behavior
type LoadBalancerConfig struct {
	Strategy              string        `yaml:"strategy"`               // round_robin, least_connections, weighted, hash
	HealthyThreshold      float64       `yaml:"healthy_threshold"`      // 0.95 = 95% success rate
	CircuitBreakerEnabled bool          `yaml:"circuit_breaker_enabled"`
	CircuitBreakerTimeout time.Duration `yaml:"circuit_breaker_timeout"`
	StickySessionEnabled  bool          `yaml:"sticky_session_enabled"`
	StickySessionTTL      time.Duration `yaml:"sticky_session_ttl"`
}

// ServiceState tracks runtime state for load balancing decisions
type ServiceState struct {
	Service         *RegisteredService
	ActiveRequests  int64
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	LastUsed        time.Time
	CircuitOpen     bool
	CircuitOpenedAt time.Time
	Weight          int
	
	// Sticky session tracking
	Sessions map[string]time.Time
	
	mutex sync.RWMutex
}

// SessionContext provides session information for sticky routing
type SessionContext struct {
	SessionID   string
	ClientID    string
	UserID      string
	Preferences map[string]interface{}
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(registry *ServiceRegistry, logger logging.Logger, metrics metrics.Metrics) *LoadBalancer {
	return &LoadBalancer{
		registry:     registry,
		logger:       logger.WithComponent("load_balancer"),
		metrics:      metrics,
		config:       defaultLoadBalancerConfig(),
		serviceState: make(map[string]*ServiceState),
	}
}

// SelectService selects the best service instance based on load balancing strategy
func (lb *LoadBalancer) SelectService(services []*RegisteredService, criteria SelectionCriteria) (*RegisteredService, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("no services available")
	}
	
	// Filter healthy services and update state
	healthyServices := lb.filterHealthyServices(services)
	if len(healthyServices) == 0 {
		return nil, fmt.Errorf("no healthy services available")
	}
	
	// Apply selection strategy
	var selected *RegisteredService
	
	switch lb.config.Strategy {
	case "round_robin":
		selected = lb.selectRoundRobin(healthyServices)
	case "least_connections":
		selected = lb.selectLeastConnections(healthyServices)
	case "weighted":
		selected = lb.selectWeighted(healthyServices)
	case "hash":
		selected = lb.selectHash(healthyServices, criteria)
	case "random":
		selected = lb.selectRandom(healthyServices)
	default:
		selected = lb.selectRoundRobin(healthyServices)
	}
	
	if selected == nil {
		return nil, fmt.Errorf("failed to select service using strategy: %s", lb.config.Strategy)
	}
	
	// Update service state
	lb.updateServiceSelection(selected)
	
	lb.logger.Debug("service_selected",
		"service_id", selected.ID,
		"strategy", lb.config.Strategy,
		"total_candidates", len(services),
		"healthy_candidates", len(healthyServices))
	
	return selected, nil
}

// filterHealthyServices filters services based on health and circuit breaker state
func (lb *LoadBalancer) filterHealthyServices(services []*RegisteredService) []*RegisteredService {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	
	var healthy []*RegisteredService
	
	for _, service := range services {
		// Basic health check
		if service.Health != HealthHealthy || service.Status != StatusActive {
			continue
		}
		
		// Circuit breaker check
		if lb.config.CircuitBreakerEnabled {
			state := lb.getOrCreateServiceState(service)
			if state.CircuitOpen {
				// Check if circuit should be closed
				if time.Since(state.CircuitOpenedAt) > lb.config.CircuitBreakerTimeout {
					state.CircuitOpen = false
					lb.logger.Info("circuit_breaker_closed",
						"service_id", service.ID,
						"timeout_duration", lb.config.CircuitBreakerTimeout)
				} else {
					continue
				}
			}
		}
		
		// Success rate check
		state := lb.getOrCreateServiceState(service)
		if state.TotalRequests > 10 { // Only check after minimum requests
			successRate := float64(state.SuccessRequests) / float64(state.TotalRequests)
			if successRate < lb.config.HealthyThreshold {
				lb.logger.Warn("service_below_health_threshold",
					"service_id", service.ID,
					"success_rate", successRate,
					"threshold", lb.config.HealthyThreshold)
				continue
			}
		}
		
		healthy = append(healthy, service)
	}
	
	return healthy
}

// selectRoundRobin implements round-robin load balancing
func (lb *LoadBalancer) selectRoundRobin(services []*RegisteredService) *RegisteredService {
	if len(services) == 0 {
		return nil
	}
	
	// Find the service that was used least recently
	var selected *RegisteredService
	var oldestUsage time.Time = time.Now()
	
	for _, service := range services {
		state := lb.getOrCreateServiceState(service)
		if state.LastUsed.Before(oldestUsage) {
			oldestUsage = state.LastUsed
			selected = service
		}
	}
	
	// If no service has been used, pick the first
	if selected == nil {
		selected = services[0]
	}
	
	return selected
}

// selectLeastConnections implements least-connections load balancing
func (lb *LoadBalancer) selectLeastConnections(services []*RegisteredService) *RegisteredService {
	if len(services) == 0 {
		return nil
	}
	
	var selected *RegisteredService
	var minConnections int64 = -1
	
	for _, service := range services {
		state := lb.getOrCreateServiceState(service)
		if minConnections == -1 || state.ActiveRequests < minConnections {
			minConnections = state.ActiveRequests
			selected = service
		}
	}
	
	return selected
}

// selectWeighted implements weighted load balancing
func (lb *LoadBalancer) selectWeighted(services []*RegisteredService) *RegisteredService {
	if len(services) == 0 {
		return nil
	}
	
	// Calculate total weight
	totalWeight := 0
	for _, service := range services {
		state := lb.getOrCreateServiceState(service)
		weight := state.Weight
		if weight <= 0 {
			weight = 1 // Default weight
		}
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return services[0]
	}
	
	// Select based on weighted random
	randValue := rand.Intn(totalWeight)
	currentWeight := 0
	
	for _, service := range services {
		state := lb.getOrCreateServiceState(service)
		weight := state.Weight
		if weight <= 0 {
			weight = 1
		}
		currentWeight += weight
		if randValue < currentWeight {
			return service
		}
	}
	
	return services[0]
}

// selectHash implements consistent hash-based load balancing
func (lb *LoadBalancer) selectHash(services []*RegisteredService, criteria SelectionCriteria) *RegisteredService {
	if len(services) == 0 {
		return nil
	}
	
	// Create hash key from criteria
	hashKey := ""
	if criteria.LoadBalancing != "" {
		hashKey = criteria.LoadBalancing
	} else if criteria.PreferredRegion != "" {
		hashKey = criteria.PreferredRegion
	} else {
		// Use a default key - perhaps client IP or session ID
		hashKey = "default"
	}
	
	// Hash the key
	hasher := fnv.New32a()
	hasher.Write([]byte(hashKey))
	hash := hasher.Sum32()
	
	// Select service based on hash
	index := int(hash) % len(services)
	return services[index]
}

// selectRandom implements random load balancing
func (lb *LoadBalancer) selectRandom(services []*RegisteredService) *RegisteredService {
	if len(services) == 0 {
		return nil
	}
	
	index := rand.Intn(len(services))
	return services[index]
}

// updateServiceSelection updates service state after selection
func (lb *LoadBalancer) updateServiceSelection(service *RegisteredService) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	state := lb.getOrCreateServiceState(service)
	state.LastUsed = time.Now()
	state.ActiveRequests++
	state.TotalRequests++
}

// RecordSuccess records a successful request completion
func (lb *LoadBalancer) RecordSuccess(service *RegisteredService, duration time.Duration) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	state := lb.getOrCreateServiceState(service)
	state.ActiveRequests--
	state.SuccessRequests++
	
	// Update service metrics
	service.Metrics.RequestCount++
	service.Metrics.LastRequestTime = time.Now()
	service.Metrics.AverageLatency = lb.updateAverageLatency(service.Metrics.AverageLatency, duration, service.Metrics.RequestCount)
	
	// Record metrics
	lb.metrics.Inc("load_balancer_requests_success_total",
		"service_id", service.ID,
		"service_type", service.Type)
	lb.metrics.Observe("load_balancer_request_duration_seconds", duration.Seconds(),
		"service_id", service.ID,
		"service_type", service.Type)
	
	lb.logger.Debug("request_completed_successfully",
		"service_id", service.ID,
		"duration", duration,
		"active_requests", state.ActiveRequests)
}

// RecordFailure records a failed request
func (lb *LoadBalancer) RecordFailure(service *RegisteredService, err error) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	state := lb.getOrCreateServiceState(service)
	state.ActiveRequests--
	state.FailedRequests++
	
	// Update service metrics
	service.Metrics.ErrorCount++
	if service.Metrics.RequestCount > 0 {
		service.Metrics.ErrorRate = float64(service.Metrics.ErrorCount) / float64(service.Metrics.RequestCount)
	}
	
	// Check if circuit breaker should be opened
	if lb.config.CircuitBreakerEnabled && state.TotalRequests > 10 {
		errorRate := float64(state.FailedRequests) / float64(state.TotalRequests)
		if errorRate > (1.0 - lb.config.HealthyThreshold) {
			state.CircuitOpen = true
			state.CircuitOpenedAt = time.Now()
			
			lb.logger.Warn("circuit_breaker_opened",
				"service_id", service.ID,
				"error_rate", errorRate,
				"threshold", 1.0-lb.config.HealthyThreshold)
		}
	}
	
	// Record metrics
	lb.metrics.Inc("load_balancer_requests_failure_total",
		"service_id", service.ID,
		"service_type", service.Type,
		"error_type", fmt.Sprintf("%T", err))
	
	lb.logger.Warn("request_failed",
		"service_id", service.ID,
		"error", err,
		"active_requests", state.ActiveRequests,
		"total_failures", state.FailedRequests)
}

// GetServiceStats returns load balancing statistics for a service
func (lb *LoadBalancer) GetServiceStats(serviceID string) *ServiceState {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	
	if state, exists := lb.serviceState[serviceID]; exists {
		// Return a copy to avoid race conditions
		stateCopy := *state
		return &stateCopy
	}
	
	return nil
}

// GetAllStats returns load balancing statistics for all services
func (lb *LoadBalancer) GetAllStats() map[string]*ServiceState {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	
	result := make(map[string]*ServiceState)
	for id, state := range lb.serviceState {
		stateCopy := *state
		result[id] = &stateCopy
	}
	
	return result
}

// ResetCircuitBreaker manually resets the circuit breaker for a service
func (lb *LoadBalancer) ResetCircuitBreaker(serviceID string) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	if state, exists := lb.serviceState[serviceID]; exists {
		state.CircuitOpen = false
		lb.logger.Info("circuit_breaker_manually_reset", "service_id", serviceID)
	}
}

// getOrCreateServiceState gets or creates service state (assumes lock is held)
func (lb *LoadBalancer) getOrCreateServiceState(service *RegisteredService) *ServiceState {
	if state, exists := lb.serviceState[service.ID]; exists {
		return state
	}
	
	state := &ServiceState{
		Service:  service,
		Weight:   1, // Default weight
		Sessions: make(map[string]time.Time),
	}
	
	lb.serviceState[service.ID] = state
	return state
}

// updateAverageLatency calculates running average latency
func (lb *LoadBalancer) updateAverageLatency(currentAvg time.Duration, newDuration time.Duration, totalRequests uint64) time.Duration {
	if totalRequests == 1 {
		return newDuration
	}
	
	// Calculate weighted average
	weight := float64(totalRequests-1) / float64(totalRequests)
	newWeight := 1.0 / float64(totalRequests)
	
	avgNanos := float64(currentAvg.Nanoseconds())*weight + float64(newDuration.Nanoseconds())*newWeight
	return time.Duration(int64(avgNanos))
}

// CleanupStaleState removes state for services that no longer exist
func (lb *LoadBalancer) CleanupStaleState() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	allServices := lb.registry.GetAllServices()
	
	// Remove state for services that no longer exist
	for serviceID := range lb.serviceState {
		if _, exists := allServices[serviceID]; !exists {
			delete(lb.serviceState, serviceID)
			lb.logger.Debug("removed_stale_service_state", "service_id", serviceID)
		}
	}
}

func defaultLoadBalancerConfig() LoadBalancerConfig {
	return LoadBalancerConfig{
		Strategy:              "round_robin",
		HealthyThreshold:      0.95,
		CircuitBreakerEnabled: true,
		CircuitBreakerTimeout: 30 * time.Second,
		StickySessionEnabled:  false,
		StickySessionTTL:      60 * time.Minute,
	}
}