package types

import (
	"encoding/json"
	"time"
)

// Protocol version constants
const (
	ProtocolVersion = "2025-03-26"
	MCPVersion      = "0.1.0"
)

// JSON-RPC 2.0 base types

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Error represents a JSON-RPC 2.0 error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// MCP-specific error codes
const (
	ErrorCodeResourceNotFound = -32001
	ErrorCodeToolNotFound     = -32002
	ErrorCodePromptNotFound   = -32003
	ErrorCodeServiceUnavailable = -32004
)

// MCP Protocol Types

// InitializeParams represents initialization parameters
type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo      `json:"clientInfo"`
}

// InitializeResult represents initialization response
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ClientCapabilities describes what the client supports
type ClientCapabilities struct {
	Sampling *SamplingCapability `json:"sampling,omitempty"`
	Roots    *RootsCapability    `json:"roots,omitempty"`
}

// ServerCapabilities describes what the server supports  
type ServerCapabilities struct {
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

// SamplingCapability describes sampling support
type SamplingCapability struct{}

// RootsCapability describes roots support
type RootsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ResourcesCapability describes resources support
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

// ToolsCapability describes tools support
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// PromptsCapability describes prompts support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// LoggingCapability describes logging support
type LoggingCapability struct {
	Level string `json:"level"`
}

// ClientInfo describes the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo describes the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Resource Types

// ResourcesListResult represents the response to resources/list
type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string            `json:"uri"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	MimeType    string            `json:"mimeType,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ResourcesReadParams represents parameters for resources/read
type ResourcesReadParams struct {
	URI string `json:"uri"`
}

// ResourcesReadResult represents the response to resources/read
type ResourcesReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent represents resource content
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 encoded
}

// Tool Types

// ToolsListResult represents the response to tools/list
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolsCallParams represents parameters for tools/call
type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolsCallResult represents the response to tools/call
type ToolsCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents tool execution result content
type ToolContent struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// Prompt Types

// PromptsListResult represents the response to prompts/list
type PromptsListResult struct {
	Prompts []Prompt `json:"prompts"`
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptsGetParams represents parameters for prompts/get
type PromptsGetParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// PromptsGetResult represents the response to prompts/get
type PromptsGetResult struct {
	Messages []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string            `json:"role"`
	Content PromptContent     `json:"content"`
}

// PromptContent represents content in a prompt message
type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Logging Types

// LoggingSetLevelParams represents parameters for logging/setLevel
type LoggingSetLevelParams struct {
	Level string `json:"level"`
}

// Progress Types

// ProgressNotification represents a progress notification
type ProgressNotification struct {
	ProgressToken interface{} `json:"progressToken"`
	Progress      float64     `json:"progress"`
	Total         float64     `json:"total,omitempty"`
}

// Internal Types for MCPEG

// ServiceError represents a service-specific error with context
type ServiceError struct {
	Code             int                    `json:"code"`
	Message          string                 `json:"message"`
	Service          string                 `json:"service"`
	Type             string                 `json:"type"`
	Details          string                 `json:"details"`
	SuggestedActions []string               `json:"suggested_actions,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
}

// AdapterRequest represents a request to a service adapter
type AdapterRequest struct {
	Method    string                 `json:"method"`
	Params    map[string]interface{} `json:"params"`
	Context   RequestContext         `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
}

// AdapterResponse represents a response from a service adapter
type AdapterResponse struct {
	Result    interface{}  `json:"result,omitempty"`
	Error     *ServiceError `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// RequestContext provides context for adapter requests
type RequestContext struct {
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
	UserAgent string            `json:"user_agent,omitempty"`
	ClientIP  string            `json:"client_ip,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// MCP Result Types

// ListResourcesResult represents the result of listing resources
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// ReadResourceResult represents the result of reading a resource
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ListToolsResult represents the result of listing tools
type ListToolsResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// CallToolResult represents the result of calling a tool
type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ListPromptsResult represents the result of listing prompts
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// GetPromptResult represents the result of getting a prompt
type GetPromptResult struct {
	Description string        `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// CompleteResult represents the result of completion
type CompleteResult struct {
	Completion CompletionResult `json:"completion"`
}

// SubscribeResult represents the result of subscribing to updates
type SubscribeResult struct{}

// UnsubscribeResult represents the result of unsubscribing from updates
type UnsubscribeResult struct{}

// LoggingLevelResult represents the result of setting logging level
type LoggingLevelResult struct{}

// CompletionResult represents completion response
type CompletionResult struct {
	Model  string      `json:"model"`
	Stop   string      `json:"stop,omitempty"`
	Values interface{} `json:"values,omitempty"`
}

// Content represents generic content
type Content struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// Implementation represents service implementation info
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}