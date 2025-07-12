package mcp

import (
	"context"
	"time"

	"github.com/osakka/mcpeg/pkg/rbac"
)

// PluginHandler defines the interface for handling plugin operations in MCP context
type PluginHandler interface {
	// InvokePlugin executes a plugin tool with the given parameters
	InvokePlugin(ctx context.Context, pluginName, toolName string, params map[string]interface{}, capabilities *rbac.ProcessedCapabilities) (*ToolResult, error)

	// GetPluginCapabilities returns the capabilities of a specific plugin filtered by user permissions
	GetPluginCapabilities(pluginName string, capabilities *rbac.ProcessedCapabilities) (*PluginCapabilities, error)

	// ListAvailablePlugins returns a list of plugins the user has access to
	ListAvailablePlugins(capabilities *rbac.ProcessedCapabilities) []string

	// GetPluginTools returns the tools available for a plugin
	GetPluginTools(pluginName string, capabilities *rbac.ProcessedCapabilities) ([]Tool, error)

	// GetPluginResources returns the resources available for a plugin
	GetPluginResources(pluginName string, capabilities *rbac.ProcessedCapabilities) ([]Resource, error)

	// GetPluginPrompts returns the prompts available for a plugin
	GetPluginPrompts(pluginName string, capabilities *rbac.ProcessedCapabilities) ([]Prompt, error)

	// HealthCheck checks if a plugin is healthy and accessible
	HealthCheck(pluginName string) (*PluginHealth, error)
}

// PluginCapabilities represents the capabilities of a plugin
type PluginCapabilities struct {
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Description string                `json:"description"`
	Tools       []Tool                `json:"tools"`
	Resources   []Resource            `json:"resources"`
	Prompts     []Prompt              `json:"prompts"`
	Permissions rbac.PluginPermission `json:"permissions"`
}

// PluginHealth represents the health status of a plugin
type PluginHealth struct {
	Name      string    `json:"name"`
	Healthy   bool      `json:"healthy"`
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
	Error     string    `json:"error,omitempty"`
}

// MCP Protocol Types (based on MCP 2025-03-26 specification)

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Category    string                 `json:"category,omitempty"`
	Examples    []ToolExample          `json:"examples,omitempty"`
}

// ToolExample provides usage examples for tools
type ToolExample struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output,omitempty"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError"`
	Meta    *ToolMeta `json:"_meta,omitempty"`
}

// Content represents different types of content in tool results
type Content interface {
	GetType() string
}

// TextContent represents text content
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (tc TextContent) GetType() string { return tc.Type }

// ImageContent represents image content
type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

func (ic ImageContent) GetType() string { return ic.Type }

// ResourceContent represents embedded resource content
type ResourceContent struct {
	Type     string   `json:"type"`
	Resource Resource `json:"resource"`
}

func (rc ResourceContent) GetType() string { return rc.Type }

// ToolMeta provides metadata about tool execution
type ToolMeta struct {
	ProgressToken string `json:"progressToken,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt template argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// JSON-RPC 2.0 Request/Response Types

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Method-Specific Types

// ToolsListRequest represents a tools/list request
type ToolsListRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  struct{}    `json:"params"`
}

// ToolsListResponse represents a tools/list response
type ToolsListResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  struct {
		Tools []Tool `json:"tools"`
	} `json:"result"`
}

// ToolsCallRequest represents a tools/call request
type ToolsCallRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	} `json:"params"`
}

// ToolsCallResponse represents a tools/call response
type ToolsCallResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  struct {
		Content []Content `json:"content"`
		IsError bool      `json:"isError"`
		Meta    *ToolMeta `json:"_meta,omitempty"`
	} `json:"result"`
}

// ResourcesListRequest represents a resources/list request
type ResourcesListRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  struct {
		Cursor string `json:"cursor,omitempty"`
	} `json:"params"`
}

// ResourcesListResponse represents a resources/list response
type ResourcesListResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  struct {
		Resources  []Resource `json:"resources"`
		NextCursor string     `json:"nextCursor,omitempty"`
	} `json:"result"`
}

// PromptsListRequest represents a prompts/list request
type PromptsListRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  struct {
		Cursor string `json:"cursor,omitempty"`
	} `json:"params"`
}

// PromptsListResponse represents a prompts/list response
type PromptsListResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  struct {
		Prompts    []Prompt `json:"prompts"`
		NextCursor string   `json:"nextCursor,omitempty"`
	} `json:"result"`
}

// Standard JSON-RPC Error Codes
const (
	// Standard JSON-RPC 2.0 errors
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603

	// MCP-specific error codes
	ErrorCodeNotFound     = -32404
	ErrorCodeForbidden    = -32403
	ErrorCodeUnauthorized = -32401
	ErrorCodeTimeout      = -32408
	ErrorCodeRateLimited  = -32429
)

// Helper functions for creating common responses

// NewErrorResponse creates a JSON-RPC error response
func NewErrorResponse(id interface{}, code int, message string, data interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewSuccessResponse creates a JSON-RPC success response
func NewSuccessResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}
