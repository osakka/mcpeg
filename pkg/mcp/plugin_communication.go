package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
	"github.com/osakka/mcpeg/pkg/metrics"
	"github.com/osakka/mcpeg/pkg/plugins"
)

// PluginCommunication manages inter-plugin communication and coordination
type PluginCommunication struct {
	pluginManager *plugins.PluginManager
	logger        logging.Logger
	metrics       metrics.Metrics
	config        PluginCommunicationConfig

	// Communication state
	messageBroker    *MessageBroker
	eventBus         *EventBus
	serviceRegistry  *ServiceRegistry
	communicationLog *CommunicationLog
	mutex            sync.RWMutex
}

// PluginCommunicationConfig configures plugin communication behavior
type PluginCommunicationConfig struct {
	// Message passing
	EnableMessagePassing   bool          `yaml:"enable_message_passing"`
	MessageTimeout         time.Duration `yaml:"message_timeout"`
	MaxMessageSize         int64         `yaml:"max_message_size"`
	MessageRetentionPeriod time.Duration `yaml:"message_retention_period"`

	// Event system
	EnableEventBus         bool          `yaml:"enable_event_bus"`
	EventBufferSize        int           `yaml:"event_buffer_size"`
	EventProcessingTimeout time.Duration `yaml:"event_processing_timeout"`

	// Service discovery
	EnableServiceDiscovery bool          `yaml:"enable_service_discovery"`
	ServiceRegistrationTTL time.Duration `yaml:"service_registration_ttl"`

	// Communication logging
	EnableCommunicationLog bool          `yaml:"enable_communication_log"`
	LogRetentionPeriod     time.Duration `yaml:"log_retention_period"`

	// Security
	RequireAuthentication     bool                `yaml:"require_authentication"`
	EnableEncryption          bool                `yaml:"enable_encryption"`
	AllowedCommunicationPairs map[string][]string `yaml:"allowed_communication_pairs"`
}

// MessageBroker handles message passing between plugins
type MessageBroker struct {
	messages    map[string][]*PluginMessage
	subscribers map[string][]MessageSubscriber
	mutex       sync.RWMutex
	logger      logging.Logger
	metrics     metrics.Metrics
}

// EventBus manages plugin events and event handling
type EventBus struct {
	events      chan *PluginEvent
	subscribers map[string][]EventSubscriber
	mutex       sync.RWMutex
	logger      logging.Logger
	metrics     metrics.Metrics
}

// ServiceRegistry manages plugin service registration and discovery
type ServiceRegistry struct {
	services map[string]*PluginService
	mutex    sync.RWMutex
	logger   logging.Logger
	metrics  metrics.Metrics
}

// CommunicationLog tracks inter-plugin communication for debugging and monitoring
type CommunicationLog struct {
	entries []CommunicationEntry
	mutex   sync.RWMutex
	logger  logging.Logger
}

// Core communication types

// PluginMessage represents a message between plugins
type PluginMessage struct {
	ID          string                 `json:"id"`
	FromPlugin  string                 `json:"from_plugin"`
	ToPlugin    string                 `json:"to_plugin"`
	MessageType string                 `json:"message_type"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   time.Time              `json:"timestamp"`
	TTL         time.Duration          `json:"ttl"`
	Priority    MessagePriority        `json:"priority"`
	Response    *PluginMessage         `json:"response,omitempty"`
	Metadata    map[string]string      `json:"metadata"`
}

// PluginEvent represents an event in the plugin system
type PluginEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  EventPriority          `json:"priority"`
	Metadata  map[string]string      `json:"metadata"`
}

// PluginService represents a service provided by a plugin
type PluginService struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Plugin       string            `json:"plugin"`
	Description  string            `json:"description"`
	Endpoints    []ServiceEndpoint `json:"endpoints"`
	Capabilities []string          `json:"capabilities"`
	Status       ServiceStatus     `json:"status"`
	RegisteredAt time.Time         `json:"registered_at"`
	LastSeen     time.Time         `json:"last_seen"`
	TTL          time.Duration     `json:"ttl"`
	Metadata     map[string]string `json:"metadata"`
}

// ServiceEndpoint represents an endpoint provided by a plugin service
type ServiceEndpoint struct {
	Name         string                 `json:"name"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	InputSchema  map[string]interface{} `json:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema"`
	Description  string                 `json:"description"`
}

// CommunicationEntry represents a log entry for plugin communication
type CommunicationEntry struct {
	ID           string                 `json:"id"`
	Type         CommunicationType      `json:"type"`
	FromPlugin   string                 `json:"from_plugin"`
	ToPlugin     string                 `json:"to_plugin"`
	Operation    string                 `json:"operation"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Enums and constants

type MessagePriority int

const (
	MessagePriorityLow MessagePriority = iota
	MessagePriorityNormal
	MessagePriorityHigh
	MessagePriorityCritical
)

type EventPriority int

const (
	EventPriorityLow EventPriority = iota
	EventPriorityNormal
	EventPriorityHigh
	EventPriorityCritical
)

type ServiceStatus int

const (
	ServiceStatusActive ServiceStatus = iota
	ServiceStatusInactive
	ServiceStatusMaintenance
	ServiceStatusError
)

type CommunicationType int

const (
	CommunicationTypeMessage CommunicationType = iota
	CommunicationTypeEvent
	CommunicationTypeServiceCall
	CommunicationTypeServiceRegistration
)

// Callback interfaces

type MessageSubscriber interface {
	OnMessage(ctx context.Context, message *PluginMessage) error
	GetSubscriptionTopics() []string
}

type EventSubscriber interface {
	OnEvent(ctx context.Context, event *PluginEvent) error
	GetEventTypes() []string
}

// NewPluginCommunication creates a new plugin communication manager
func NewPluginCommunication(
	pluginManager *plugins.PluginManager,
	logger logging.Logger,
	metrics metrics.Metrics,
) *PluginCommunication {
	config := defaultPluginCommunicationConfig()

	pc := &PluginCommunication{
		pluginManager: pluginManager,
		logger:        logger.WithComponent("plugin_communication"),
		metrics:       metrics,
		config:        config,
	}

	// Initialize components
	if config.EnableMessagePassing {
		pc.messageBroker = NewMessageBroker(logger, metrics)
	}

	if config.EnableEventBus {
		pc.eventBus = NewEventBus(config.EventBufferSize, logger, metrics)
	}

	if config.EnableServiceDiscovery {
		pc.serviceRegistry = NewServiceRegistry(logger, metrics)
	}

	if config.EnableCommunicationLog {
		pc.communicationLog = NewCommunicationLog(logger)
	}

	return pc
}

// Message Broker Implementation

func NewMessageBroker(logger logging.Logger, metrics metrics.Metrics) *MessageBroker {
	return &MessageBroker{
		messages:    make(map[string][]*PluginMessage),
		subscribers: make(map[string][]MessageSubscriber),
		logger:      logger.WithComponent("message_broker"),
		metrics:     metrics,
	}
}

// SendMessage sends a message from one plugin to another
func (pc *PluginCommunication) SendMessage(ctx context.Context, fromPlugin, toPlugin string, messageType string, payload map[string]interface{}) (*PluginMessage, error) {
	if !pc.config.EnableMessagePassing || pc.messageBroker == nil {
		return nil, fmt.Errorf("message passing is not enabled")
	}

	// Check if communication is allowed
	if !pc.isCommunicationAllowed(fromPlugin, toPlugin) {
		return nil, fmt.Errorf("communication not allowed between %s and %s", fromPlugin, toPlugin)
	}

	message := &PluginMessage{
		ID:          generateMessageID(),
		FromPlugin:  fromPlugin,
		ToPlugin:    toPlugin,
		MessageType: messageType,
		Payload:     payload,
		Timestamp:   time.Now(),
		TTL:         pc.config.MessageTimeout,
		Priority:    MessagePriorityNormal,
		Metadata:    make(map[string]string),
	}

	pc.logger.Debug("plugin_message_sending",
		"message_id", message.ID,
		"from_plugin", fromPlugin,
		"to_plugin", toPlugin,
		"message_type", messageType)

	// Store message for retrieval
	pc.messageBroker.mutex.Lock()
	if pc.messageBroker.messages[toPlugin] == nil {
		pc.messageBroker.messages[toPlugin] = make([]*PluginMessage, 0)
	}
	pc.messageBroker.messages[toPlugin] = append(pc.messageBroker.messages[toPlugin], message)
	pc.messageBroker.mutex.Unlock()

	// Notify subscribers
	pc.notifyMessageSubscribers(ctx, message)

	// Log communication
	if pc.config.EnableCommunicationLog && pc.communicationLog != nil {
		pc.logCommunication(CommunicationTypeMessage, fromPlugin, toPlugin, "send_message", true, "", time.Since(message.Timestamp), nil)
	}

	pc.metrics.Inc("plugin_messages_sent", "from_plugin", fromPlugin, "to_plugin", toPlugin, "message_type", messageType)

	pc.logger.Info("plugin_message_sent",
		"message_id", message.ID,
		"from_plugin", fromPlugin,
		"to_plugin", toPlugin,
		"message_type", messageType)

	return message, nil
}

// ReceiveMessages retrieves messages for a plugin
func (pc *PluginCommunication) ReceiveMessages(ctx context.Context, pluginName string) ([]*PluginMessage, error) {
	if !pc.config.EnableMessagePassing || pc.messageBroker == nil {
		return nil, fmt.Errorf("message passing is not enabled")
	}

	pc.messageBroker.mutex.Lock()
	defer pc.messageBroker.mutex.Unlock()

	messages := pc.messageBroker.messages[pluginName]
	if messages == nil {
		return []*PluginMessage{}, nil
	}

	// Filter out expired messages
	var validMessages []*PluginMessage
	now := time.Now()
	for _, msg := range messages {
		if now.Sub(msg.Timestamp) < msg.TTL {
			validMessages = append(validMessages, msg)
		}
	}

	// Clear the message queue for this plugin
	pc.messageBroker.messages[pluginName] = make([]*PluginMessage, 0)

	pc.metrics.Inc("plugin_messages_received", "plugin", pluginName, "count", fmt.Sprintf("%d", len(validMessages)))

	pc.logger.Debug("plugin_messages_retrieved",
		"plugin", pluginName,
		"message_count", len(validMessages))

	return validMessages, nil
}

// Event Bus Implementation

func NewEventBus(bufferSize int, logger logging.Logger, metrics metrics.Metrics) *EventBus {
	return &EventBus{
		events:      make(chan *PluginEvent, bufferSize),
		subscribers: make(map[string][]EventSubscriber),
		logger:      logger.WithComponent("event_bus"),
		metrics:     metrics,
	}
}

// PublishEvent publishes an event to the event bus
func (pc *PluginCommunication) PublishEvent(ctx context.Context, eventType, source string, data map[string]interface{}) error {
	if !pc.config.EnableEventBus || pc.eventBus == nil {
		return fmt.Errorf("event bus is not enabled")
	}

	event := &PluginEvent{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    source,
		Data:      data,
		Timestamp: time.Now(),
		Priority:  EventPriorityNormal,
		Metadata:  make(map[string]string),
	}

	select {
	case pc.eventBus.events <- event:
		pc.metrics.Inc("plugin_events_published", "event_type", eventType, "source", source)
		pc.logger.Debug("plugin_event_published",
			"event_id", event.ID,
			"event_type", eventType,
			"source", source)
		return nil
	case <-time.After(pc.config.EventProcessingTimeout):
		pc.metrics.Inc("plugin_events_publish_timeout", "event_type", eventType, "source", source)
		return fmt.Errorf("timeout publishing event")
	}
}

// SubscribeToEvents subscribes a plugin to specific event types
func (pc *PluginCommunication) SubscribeToEvents(pluginName string, eventTypes []string, subscriber EventSubscriber) error {
	if !pc.config.EnableEventBus || pc.eventBus == nil {
		return fmt.Errorf("event bus is not enabled")
	}

	pc.eventBus.mutex.Lock()
	defer pc.eventBus.mutex.Unlock()

	for _, eventType := range eventTypes {
		if pc.eventBus.subscribers[eventType] == nil {
			pc.eventBus.subscribers[eventType] = make([]EventSubscriber, 0)
		}
		pc.eventBus.subscribers[eventType] = append(pc.eventBus.subscribers[eventType], subscriber)
	}

	pc.logger.Info("plugin_event_subscription_added",
		"plugin", pluginName,
		"event_types", eventTypes)

	return nil
}

// Service Registry Implementation

func NewServiceRegistry(logger logging.Logger, metrics metrics.Metrics) *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*PluginService),
		logger:   logger.WithComponent("service_registry"),
		metrics:  metrics,
	}
}

// RegisterService registers a service provided by a plugin
func (pc *PluginCommunication) RegisterService(ctx context.Context, service *PluginService) error {
	if !pc.config.EnableServiceDiscovery || pc.serviceRegistry == nil {
		return fmt.Errorf("service discovery is not enabled")
	}

	service.RegisteredAt = time.Now()
	service.LastSeen = time.Now()
	service.Status = ServiceStatusActive

	pc.serviceRegistry.mutex.Lock()
	pc.serviceRegistry.services[service.ID] = service
	pc.serviceRegistry.mutex.Unlock()

	pc.metrics.Inc("plugin_services_registered", "plugin", service.Plugin, "service", service.Name)

	pc.logger.Info("plugin_service_registered",
		"service_id", service.ID,
		"service_name", service.Name,
		"plugin", service.Plugin,
		"endpoints", len(service.Endpoints))

	// Log communication
	if pc.config.EnableCommunicationLog && pc.communicationLog != nil {
		pc.logCommunication(CommunicationTypeServiceRegistration, service.Plugin, "", "register_service", true, "", 0, map[string]interface{}{
			"service_id":   service.ID,
			"service_name": service.Name,
		})
	}

	return nil
}

// DiscoverServices discovers services provided by other plugins
func (pc *PluginCommunication) DiscoverServices(ctx context.Context, pluginName string, capabilities []string) ([]*PluginService, error) {
	if !pc.config.EnableServiceDiscovery || pc.serviceRegistry == nil {
		return nil, fmt.Errorf("service discovery is not enabled")
	}

	pc.serviceRegistry.mutex.RLock()
	defer pc.serviceRegistry.mutex.RUnlock()

	var matchingServices []*PluginService
	for _, service := range pc.serviceRegistry.services {
		// Skip services from the same plugin
		if service.Plugin == pluginName {
			continue
		}

		// Check if service matches required capabilities
		if pc.serviceMatchesCapabilities(service, capabilities) {
			matchingServices = append(matchingServices, service)
		}
	}

	pc.metrics.Inc("plugin_services_discovered", "plugin", pluginName, "matching_services", fmt.Sprintf("%d", len(matchingServices)))

	pc.logger.Debug("plugin_services_discovered",
		"plugin", pluginName,
		"required_capabilities", capabilities,
		"matching_services", len(matchingServices))

	return matchingServices, nil
}

// CallService calls a service provided by another plugin
func (pc *PluginCommunication) CallService(ctx context.Context, fromPlugin, serviceID, endpoint string, params map[string]interface{}) (interface{}, error) {
	start := time.Now()

	if !pc.config.EnableServiceDiscovery || pc.serviceRegistry == nil {
		return nil, fmt.Errorf("service discovery is not enabled")
	}

	pc.serviceRegistry.mutex.RLock()
	service, exists := pc.serviceRegistry.services[serviceID]
	pc.serviceRegistry.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	// Check if communication is allowed
	if !pc.isCommunicationAllowed(fromPlugin, service.Plugin) {
		return nil, fmt.Errorf("communication not allowed between %s and %s", fromPlugin, service.Plugin)
	}

	// Find the endpoint
	var targetEndpoint *ServiceEndpoint
	for _, ep := range service.Endpoints {
		if ep.Name == endpoint {
			targetEndpoint = &ep
			break
		}
	}

	if targetEndpoint == nil {
		return nil, fmt.Errorf("endpoint not found: %s", endpoint)
	}

	pc.logger.Debug("plugin_service_call_started",
		"from_plugin", fromPlugin,
		"to_plugin", service.Plugin,
		"service_id", serviceID,
		"endpoint", endpoint)

	// Call the target plugin's service
	targetPlugin, exists := pc.pluginManager.GetPlugin(service.Plugin)
	if !exists {
		return nil, fmt.Errorf("target plugin not found: %s", service.Plugin)
	}

	// Convert params to JSON for plugin call
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Use the endpoint path as the tool name
	result, err := targetPlugin.CallTool(ctx, targetEndpoint.Path, paramsJSON)

	duration := time.Since(start)
	success := err == nil

	// Log communication
	if pc.config.EnableCommunicationLog && pc.communicationLog != nil {
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		pc.logCommunication(CommunicationTypeServiceCall, fromPlugin, service.Plugin, endpoint, success, errorMsg, duration, map[string]interface{}{
			"service_id": serviceID,
			"endpoint":   endpoint,
		})
	}

	pc.metrics.Inc("plugin_service_calls", "from_plugin", fromPlugin, "to_plugin", service.Plugin, "success", fmt.Sprintf("%t", success))
	pc.metrics.Observe("plugin_service_call_duration", duration.Seconds(), "from_plugin", fromPlugin, "to_plugin", service.Plugin)

	if err != nil {
		pc.logger.Warn("plugin_service_call_failed",
			"from_plugin", fromPlugin,
			"to_plugin", service.Plugin,
			"service_id", serviceID,
			"endpoint", endpoint,
			"error", err,
			"duration", duration)
		return nil, err
	}

	pc.logger.Info("plugin_service_call_completed",
		"from_plugin", fromPlugin,
		"to_plugin", service.Plugin,
		"service_id", serviceID,
		"endpoint", endpoint,
		"duration", duration)

	return result, nil
}

// Helper methods

func (pc *PluginCommunication) isCommunicationAllowed(fromPlugin, toPlugin string) bool {
	if !pc.config.RequireAuthentication {
		return true
	}

	if allowedTargets, exists := pc.config.AllowedCommunicationPairs[fromPlugin]; exists {
		for _, target := range allowedTargets {
			if target == toPlugin || target == "*" {
				return true
			}
		}
		return false
	}

	// If no specific rules exist, allow communication
	return true
}

func (pc *PluginCommunication) serviceMatchesCapabilities(service *PluginService, requiredCapabilities []string) bool {
	if len(requiredCapabilities) == 0 {
		return true
	}

	for _, required := range requiredCapabilities {
		found := false
		for _, capability := range service.Capabilities {
			if capability == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (pc *PluginCommunication) notifyMessageSubscribers(ctx context.Context, message *PluginMessage) {
	if pc.messageBroker == nil {
		return
	}

	pc.messageBroker.mutex.RLock()
	subscribers := pc.messageBroker.subscribers[message.MessageType]
	pc.messageBroker.mutex.RUnlock()

	for _, subscriber := range subscribers {
		go func(sub MessageSubscriber) {
			if err := sub.OnMessage(ctx, message); err != nil {
				pc.logger.Warn("message_subscriber_error",
					"message_id", message.ID,
					"error", err)
			}
		}(subscriber)
	}
}

func (pc *PluginCommunication) logCommunication(commType CommunicationType, fromPlugin, toPlugin, operation string, success bool, errorMessage string, duration time.Duration, metadata map[string]interface{}) {
	if pc.communicationLog == nil {
		return
	}

	entry := CommunicationEntry{
		ID:           generateCommunicationID(),
		Type:         commType,
		FromPlugin:   fromPlugin,
		ToPlugin:     toPlugin,
		Operation:    operation,
		Success:      success,
		ErrorMessage: errorMessage,
		Duration:     duration,
		Timestamp:    time.Now(),
		Metadata:     metadata,
	}

	pc.communicationLog.mutex.Lock()
	pc.communicationLog.entries = append(pc.communicationLog.entries, entry)
	pc.communicationLog.mutex.Unlock()
}

// Communication Log Implementation

func NewCommunicationLog(logger logging.Logger) *CommunicationLog {
	return &CommunicationLog{
		entries: make([]CommunicationEntry, 0),
		logger:  logger.WithComponent("communication_log"),
	}
}

// GetCommunicationLog returns recent communication entries
func (pc *PluginCommunication) GetCommunicationLog(ctx context.Context, limit int) ([]CommunicationEntry, error) {
	if !pc.config.EnableCommunicationLog || pc.communicationLog == nil {
		return nil, fmt.Errorf("communication logging is not enabled")
	}

	pc.communicationLog.mutex.RLock()
	defer pc.communicationLog.mutex.RUnlock()

	entries := pc.communicationLog.entries
	if limit > 0 && len(entries) > limit {
		// Return the most recent entries
		start := len(entries) - limit
		entries = entries[start:]
	}

	return entries, nil
}

// ID generators
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

func generateCommunicationID() string {
	return fmt.Sprintf("comm_%d", time.Now().UnixNano())
}

func defaultPluginCommunicationConfig() PluginCommunicationConfig {
	return PluginCommunicationConfig{
		EnableMessagePassing:      true,
		MessageTimeout:            30 * time.Second,
		MaxMessageSize:            1024 * 1024, // 1MB
		MessageRetentionPeriod:    1 * time.Hour,
		EnableEventBus:            true,
		EventBufferSize:           1000,
		EventProcessingTimeout:    10 * time.Second,
		EnableServiceDiscovery:    true,
		ServiceRegistrationTTL:    1 * time.Hour,
		EnableCommunicationLog:    true,
		LogRetentionPeriod:        24 * time.Hour,
		RequireAuthentication:     false,
		EnableEncryption:          false,
		AllowedCommunicationPairs: make(map[string][]string),
	}
}
