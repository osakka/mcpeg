package context

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/osakka/mcpeg/pkg/logging"
)

// HTTPContextPropagator handles context propagation over HTTP
type HTTPContextPropagator struct {
	contextManager *ContextManager
	logger         logging.Logger
	config         HTTPPropagationConfig
}

// HTTPPropagationConfig configures HTTP context propagation
type HTTPPropagationConfig struct {
	// Header names for context propagation
	RequestIDHeader     string `yaml:"request_id_header"`
	CorrelationIDHeader string `yaml:"correlation_id_header"`
	TraceIDHeader       string `yaml:"trace_id_header"`
	SpanIDHeader        string `yaml:"span_id_header"`
	UserIDHeader        string `yaml:"user_id_header"`
	SessionIDHeader     string `yaml:"session_id_header"`
	TimeoutHeader       string `yaml:"timeout_header"`
	
	// Custom headers to propagate
	CustomHeaders []string `yaml:"custom_headers"`
	
	// Security settings
	SanitizeHeaders bool     `yaml:"sanitize_headers"`
	AllowedOrigins  []string `yaml:"allowed_origins"`
	
	// Generation settings
	GenerateIDs bool `yaml:"generate_ids"`
	IDPrefix    string `yaml:"id_prefix"`
}

// NewHTTPContextPropagator creates a new HTTP context propagator
func NewHTTPContextPropagator(contextManager *ContextManager, logger logging.Logger) *HTTPContextPropagator {
	return &HTTPContextPropagator{
		contextManager: contextManager,
		logger:         logger.WithComponent("http_context_propagator"),
		config:         defaultHTTPPropagationConfig(),
	}
}

// ExtractFromRequest extracts context from HTTP request headers
func (h *HTTPContextPropagator) ExtractFromRequest(req *http.Request) context.Context {
	ctx := req.Context()
	
	// Extract request context
	reqCtx := &RequestContext{
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Extract standard headers
	reqCtx.RequestID = h.getHeader(req, h.config.RequestIDHeader)
	reqCtx.CorrelationID = h.getHeader(req, h.config.CorrelationIDHeader)
	reqCtx.TraceID = h.getHeader(req, h.config.TraceIDHeader)
	reqCtx.SpanID = h.getHeader(req, h.config.SpanIDHeader)
	reqCtx.UserID = h.getHeader(req, h.config.UserIDHeader)
	reqCtx.SessionID = h.getHeader(req, h.config.SessionIDHeader)
	
	// Generate missing IDs if configured
	if h.config.GenerateIDs {
		if reqCtx.RequestID == "" {
			reqCtx.RequestID = h.generateID("req")
		}
		if reqCtx.CorrelationID == "" {
			reqCtx.CorrelationID = h.generateID("cor")
		}
		if reqCtx.TraceID == "" {
			reqCtx.TraceID = h.generateID("trc")
		}
	}
	
	// Extract timeout
	if timeoutStr := h.getHeader(req, h.config.TimeoutHeader); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			reqCtx.Timeout = timeout
		}
	}
	
	// Extract client information
	reqCtx.ClientInfo = &ClientInfo{
		UserAgent: req.UserAgent(),
		IPAddress: h.extractClientIP(req),
		Headers:   h.extractRelevantHeaders(req),
	}
	
	// Extract client info from headers if present
	if clientName := h.getHeader(req, "X-Client-Name"); clientName != "" {
		reqCtx.ClientInfo.Name = clientName
	}
	if clientVersion := h.getHeader(req, "X-Client-Version"); clientVersion != "" {
		reqCtx.ClientInfo.Version = clientVersion
	}
	
	// Add custom metadata
	for _, headerName := range h.config.CustomHeaders {
		if value := h.getHeader(req, headerName); value != "" {
			reqCtx.Metadata[strings.ToLower(headerName)] = value
		}
	}
	
	// Add request metadata
	reqCtx.Metadata["method"] = req.Method
	reqCtx.Metadata["url"] = req.URL.String()
	reqCtx.Metadata["content_length"] = req.ContentLength
	
	// Apply context to request context
	ctx = h.contextManager.WithRequestContext(ctx, reqCtx)
	
	h.logger.Debug("context_extracted_from_request",
		"request_id", reqCtx.RequestID,
		"correlation_id", reqCtx.CorrelationID,
		"trace_id", reqCtx.TraceID,
		"client_ip", reqCtx.ClientInfo.IPAddress,
		"user_agent", reqCtx.ClientInfo.UserAgent,
		"method", req.Method,
		"url", req.URL.String())
	
	return ctx
}

// InjectIntoRequest injects context into HTTP request headers
func (h *HTTPContextPropagator) InjectIntoRequest(ctx context.Context, req *http.Request) {
	reqCtx := h.contextManager.GetRequestContext(ctx)
	
	// Inject standard headers
	if reqCtx.RequestID != "" {
		req.Header.Set(h.config.RequestIDHeader, reqCtx.RequestID)
	}
	if reqCtx.CorrelationID != "" {
		req.Header.Set(h.config.CorrelationIDHeader, reqCtx.CorrelationID)
	}
	if reqCtx.TraceID != "" {
		req.Header.Set(h.config.TraceIDHeader, reqCtx.TraceID)
	}
	if reqCtx.SpanID != "" {
		req.Header.Set(h.config.SpanIDHeader, reqCtx.SpanID)
	}
	if reqCtx.UserID != "" {
		req.Header.Set(h.config.UserIDHeader, reqCtx.UserID)
	}
	if reqCtx.SessionID != "" {
		req.Header.Set(h.config.SessionIDHeader, reqCtx.SessionID)
	}
	
	// Inject timeout if present
	if reqCtx.Timeout > 0 {
		req.Header.Set(h.config.TimeoutHeader, reqCtx.Timeout.String())
	}
	
	// Inject client info
	if reqCtx.ClientInfo != nil {
		if reqCtx.ClientInfo.Name != "" {
			req.Header.Set("X-Client-Name", reqCtx.ClientInfo.Name)
		}
		if reqCtx.ClientInfo.Version != "" {
			req.Header.Set("X-Client-Version", reqCtx.ClientInfo.Version)
		}
	}
	
	// Inject service context
	svcCtx := h.contextManager.GetServiceContext(ctx)
	if svcCtx.ServiceName != "" {
		req.Header.Set("X-Service-Name", svcCtx.ServiceName)
	}
	if svcCtx.ServiceVersion != "" {
		req.Header.Set("X-Service-Version", svcCtx.ServiceVersion)
	}
	if svcCtx.Operation != "" {
		req.Header.Set("X-Operation", svcCtx.Operation)
	}
	
	h.logger.Debug("context_injected_into_request",
		"request_id", reqCtx.RequestID,
		"correlation_id", reqCtx.CorrelationID,
		"trace_id", reqCtx.TraceID,
		"service", svcCtx.ServiceName,
		"operation", svcCtx.Operation,
		"url", req.URL.String())
}

// InjectIntoResponse injects context into HTTP response headers
func (h *HTTPContextPropagator) InjectIntoResponse(ctx context.Context, w http.ResponseWriter) {
	reqCtx := h.contextManager.GetRequestContext(ctx)
	
	// Inject response headers for tracking
	if reqCtx.RequestID != "" {
		w.Header().Set(h.config.RequestIDHeader, reqCtx.RequestID)
	}
	if reqCtx.CorrelationID != "" {
		w.Header().Set(h.config.CorrelationIDHeader, reqCtx.CorrelationID)
	}
	if reqCtx.TraceID != "" {
		w.Header().Set(h.config.TraceIDHeader, reqCtx.TraceID)
	}
	
	// Add timing information
	if !reqCtx.StartTime.IsZero() {
		duration := time.Since(reqCtx.StartTime)
		w.Header().Set("X-Response-Time", duration.String())
		w.Header().Set("X-Response-Time-Ms", strconv.FormatFloat(duration.Seconds()*1000, 'f', 2, 64))
	}
	
	// Add service information
	svcCtx := h.contextManager.GetServiceContext(ctx)
	if svcCtx.ServiceName != "" {
		w.Header().Set("X-Service-Name", svcCtx.ServiceName)
	}
	if svcCtx.ServiceVersion != "" {
		w.Header().Set("X-Service-Version", svcCtx.ServiceVersion)
	}
}

// Middleware returns HTTP middleware for automatic context propagation
func (h *HTTPContextPropagator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract context from request
		ctx := h.ExtractFromRequest(r)
		
		// Create new request with enriched context
		r = r.WithContext(ctx)
		
		// Add response headers before calling next handler
		h.InjectIntoResponse(ctx, w)
		
		// Call next handler
		next.ServeHTTP(w, r)
		
		// Log request completion
		reqCtx := h.contextManager.GetRequestContext(ctx)
		duration := time.Since(reqCtx.StartTime)
		
		h.logger.Info("http_request_completed",
			"request_id", reqCtx.RequestID,
			"correlation_id", reqCtx.CorrelationID,
			"method", r.Method,
			"url", r.URL.String(),
			"duration", duration,
			"client_ip", reqCtx.ClientInfo.IPAddress,
			"user_agent", reqCtx.ClientInfo.UserAgent)
	})
}

// extractClientIP extracts the real client IP from request
func (h *HTTPContextPropagator) extractClientIP(req *http.Request) string {
	// Check common headers for client IP
	headers := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Client-IP",
		"CF-Connecting-IP", // Cloudflare
		"True-Client-IP",   // Akamai
	}
	
	for _, header := range headers {
		if ip := req.Header.Get(header); ip != "" {
			// X-Forwarded-For can contain multiple IPs, take the first one
			if strings.Contains(ip, ",") {
				ip = strings.TrimSpace(strings.Split(ip, ",")[0])
			}
			if ip != "" && ip != "unknown" {
				return ip
			}
		}
	}
	
	// Fallback to RemoteAddr
	ip := req.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	
	return ip
}

// extractRelevantHeaders extracts headers relevant for debugging/tracing
func (h *HTTPContextPropagator) extractRelevantHeaders(req *http.Request) map[string]string {
	headers := make(map[string]string)
	
	// Standard headers to capture
	relevantHeaders := []string{
		"Content-Type",
		"Accept",
		"Authorization", // Will be sanitized if needed
		"X-API-Key",     // Will be sanitized if needed
		"User-Agent",
		"Referer",
		"Origin",
	}
	
	for _, headerName := range relevantHeaders {
		if value := req.Header.Get(headerName); value != "" {
			if h.config.SanitizeHeaders && h.isSensitiveHeader(headerName) {
				headers[headerName] = h.sanitizeHeaderValue(value)
			} else {
				headers[headerName] = value
			}
		}
	}
	
	// Add custom headers
	for _, headerName := range h.config.CustomHeaders {
		if value := req.Header.Get(headerName); value != "" {
			headers[headerName] = value
		}
	}
	
	return headers
}

// getHeader gets header value with case-insensitive lookup
func (h *HTTPContextPropagator) getHeader(req *http.Request, headerName string) string {
	return req.Header.Get(headerName)
}

// generateID generates a unique ID with prefix
func (h *HTTPContextPropagator) generateID(prefix string) string {
	// Simple ID generation - in production, use proper UUID or nanoid
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%s_%d", h.config.IDPrefix, prefix, timestamp)
}

// isSensitiveHeader checks if a header contains sensitive information
func (h *HTTPContextPropagator) isSensitiveHeader(headerName string) bool {
	sensitiveHeaders := []string{
		"authorization",
		"x-api-key",
		"x-auth-token",
		"cookie",
		"set-cookie",
		"x-forwarded-authorization",
	}
	
	headerLower := strings.ToLower(headerName)
	for _, sensitive := range sensitiveHeaders {
		if headerLower == sensitive || strings.Contains(headerLower, "auth") || strings.Contains(headerLower, "token") {
			return true
		}
	}
	
	return false
}

// sanitizeHeaderValue sanitizes sensitive header values
func (h *HTTPContextPropagator) sanitizeHeaderValue(value string) string {
	if len(value) == 0 {
		return ""
	}
	
	// For Bearer tokens, show only the type and last 4 characters
	if strings.HasPrefix(value, "Bearer ") {
		token := value[7:]
		if len(token) > 4 {
			return fmt.Sprintf("Bearer ...%s", token[len(token)-4:])
		}
		return "Bearer ****"
	}
	
	// For API keys, show only first and last 4 characters
	if len(value) > 8 {
		return fmt.Sprintf("%s...%s", value[:4], value[len(value)-4:])
	}
	
	// For short values, mask completely
	return "****"
}

func defaultHTTPPropagationConfig() HTTPPropagationConfig {
	return HTTPPropagationConfig{
		RequestIDHeader:     "X-Request-ID",
		CorrelationIDHeader: "X-Correlation-ID",
		TraceIDHeader:       "X-Trace-ID",
		SpanIDHeader:        "X-Span-ID",
		UserIDHeader:        "X-User-ID",
		SessionIDHeader:     "X-Session-ID",
		TimeoutHeader:       "X-Timeout",
		CustomHeaders: []string{
			"X-Client-Name",
			"X-Client-Version",
			"X-Feature-Flags",
		},
		SanitizeHeaders: true,
		AllowedOrigins: []string{
			"https://claude.ai",
			"https://*.anthropic.com",
		},
		GenerateIDs: true,
		IDPrefix:    "mcpeg",
	}
}

import "fmt"