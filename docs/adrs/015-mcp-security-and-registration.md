# ADR-015: MCP Security and Registration Requirements

## Status

Proposed

## Context

For MCPEG to work appropriately in production, we need to understand and implement proper MCP security requirements including:
1. SSL/TLS integration for secure transport
2. Authentication requirements between clients and servers
3. MCP service registration and discovery mechanisms
4. Bar-raising security beyond basic MCP specification

The MCP specification provides transport security guidance but doesn't mandate specific authentication mechanisms, leaving implementation details to the server.

## Research Findings

### MCP Transport Security Requirements

From MCP documentation:
- **Mandatory TLS**: "Use TLS for network transport" in production
- **Origin Validation**: Validate Origin headers to prevent DNS rebinding attacks
- **Localhost Binding**: Bind local servers only to localhost (127.0.0.1)
- **Rate Limiting**: Implement proper rate limiting
- **Input Sanitization**: Validate and sanitize all input data

### MCP Registration Process

The MCP protocol uses an initialization handshake:
1. Client sends `initialize` request with:
   - `protocolVersion` - MCP version supported
   - `clientInfo` - Client name and version
   - `capabilities` - Client's supported features

2. Server responds with:
   - `protocolVersion` - Server's MCP version
   - `serverInfo` - Server name and version
   - `capabilities` - Server's available features

**Key Finding**: MCP doesn't define authentication - it's left to implementation!

## Decision

We will implement a **bar-raising security framework** that goes beyond basic MCP requirements:

### 1. Multi-Layer TLS Security

**Transport Layer Security:**
```yaml
# config/mcpeg.yaml
transports:
  stdio:
    enabled: true
    # stdio is inherently local, no TLS needed
    
  http:
    enabled: true
    tls:
      enabled: true
      cert_file: "${TLS_CERT_PATH}"
      key_file: "${TLS_KEY_PATH}"
      min_version: "1.3"                 # TLS 1.3 minimum
      cipher_suites:                     # Strong ciphers only
        - "TLS_AES_256_GCM_SHA384"
        - "TLS_CHACHA20_POLY1305_SHA256"
      client_auth: "require"             # Mutual TLS
      client_ca_file: "${CLIENT_CA_PATH}"
      
    security:
      bind_address: "127.0.0.1"          # Localhost only
      cors:
        enabled: true
        allowed_origins: ["https://claude.ai"]
        allowed_methods: ["POST"]
        allowed_headers: ["Content-Type", "Authorization"]
      
      headers:
        strict_transport_security: "max-age=31536000; includeSubDomains"
        content_type_options: "nosniff"
        frame_options: "DENY"
        referrer_policy: "strict-origin-when-cross-origin"
```

### 2. Enhanced Authentication Framework

**Multi-Tier Authentication:**
```go
// pkg/security/auth.go
type AuthLevel string

const (
    AuthNone     AuthLevel = "none"      // Development only
    AuthAPIKey   AuthLevel = "api_key"   // API key authentication
    AuthJWT      AuthLevel = "jwt"       // JWT tokens
    AuthMTLS     AuthLevel = "mtls"      // Mutual TLS certificates
    AuthOAuth2   AuthLevel = "oauth2"    // OAuth2 flow
)

type MCPAuthenticator struct {
    level    AuthLevel
    provider AuthProvider
    logger   logging.Logger
    metrics  metrics.Metrics
}

// Enhanced MCP initialization with authentication
type EnhancedInitializeParams struct {
    types.InitializeParams              // Standard MCP fields
    
    // Security extensions
    Authentication *AuthenticationInfo  `json:"authentication,omitempty"`
    Security       *SecurityContext     `json:"security,omitempty"`
}

type AuthenticationInfo struct {
    Method      string            `json:"method"`      // "api_key", "jwt", "mtls"
    Credentials map[string]string `json:"credentials"` // Method-specific credentials
    ClientID    string            `json:"client_id"`   // Unique client identifier
}

type SecurityContext struct {
    IPAddress    string   `json:"ip_address"`
    UserAgent    string   `json:"user_agent"`
    Permissions  []string `json:"permissions"`
    RateLimit    *RateLimit `json:"rate_limit,omitempty"`
}
```

### 3. Advanced MCP Registration & Discovery

**Service Registry with Security:**
```go
// internal/registry/mcp_registry.go
type MCPRegistry struct {
    services    map[string]*RegisteredService
    security    SecurityManager
    discovery   DiscoveryService
    health      HealthManager
}

type RegisteredService struct {
    ID            string                 `json:"id"`
    Name          string                 `json:"name"`
    Version       string                 `json:"version"`
    Endpoint      string                 `json:"endpoint"`
    Capabilities  types.ServerCapabilities `json:"capabilities"`
    Security      ServiceSecurity        `json:"security"`
    Health        HealthStatus           `json:"health"`
    RegisteredAt  time.Time             `json:"registered_at"`
    LastSeen      time.Time             `json:"last_seen"`
}

type ServiceSecurity struct {
    AuthRequired     bool     `json:"auth_required"`
    TLSRequired      bool     `json:"tls_required"`
    AllowedClients   []string `json:"allowed_clients"`
    RequiredScopes   []string `json:"required_scopes"`
    RateLimit        *RateLimit `json:"rate_limit"`
}

// Registration endpoint with enhanced security
func (r *MCPRegistry) RegisterService(ctx context.Context, req RegisterServiceRequest) error {
    // Validate client authentication
    if err := r.security.ValidateClient(ctx, req.ClientAuth); err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }
    
    // Verify service capabilities
    if err := r.validateCapabilities(req.Capabilities); err != nil {
        return fmt.Errorf("invalid capabilities: %w", err)
    }
    
    // Perform health check
    if err := r.health.CheckService(ctx, req.Endpoint); err != nil {
        return fmt.Errorf("health check failed: %w", err)
    }
    
    // Register with security context
    service := &RegisteredService{
        ID:           generateServiceID(),
        Name:         req.Name,
        Version:      req.Version,
        Endpoint:     req.Endpoint,
        Capabilities: req.Capabilities,
        Security:     req.Security,
        RegisteredAt: time.Now(),
    }
    
    r.services[service.ID] = service
    
    r.logger.Info("service_registered",
        "service_id", service.ID,
        "name", service.Name,
        "endpoint", service.Endpoint,
        "auth_required", service.Security.AuthRequired,
        "tls_required", service.Security.TLSRequired)
    
    return nil
}
```

### 4. Security-Enhanced MCP Handshake

**Extended Initialization Process:**
```go
// Enhanced initialization with security validation
func (s *MCPServer) HandleInitialize(ctx context.Context, req EnhancedInitializeParams) (*types.InitializeResult, error) {
    // Phase 1: Standard MCP validation
    if err := s.validateMCPVersion(req.ProtocolVersion); err != nil {
        return nil, fmt.Errorf("unsupported protocol version: %w", err)
    }
    
    // Phase 2: Security authentication
    if s.config.Security.AuthRequired {
        principal, err := s.authenticator.Authenticate(ctx, req.Authentication)
        if err != nil {
            s.logger.Error("authentication_failed",
                "client_id", req.Authentication.ClientID,
                "method", req.Authentication.Method,
                "ip_address", req.Security.IPAddress,
                "error", err)
            return nil, fmt.Errorf("authentication failed: %w", err)
        }
        
        // Store authenticated principal in context
        ctx = WithPrincipal(ctx, principal)
    }
    
    // Phase 3: Authorization check
    if err := s.authorizer.Authorize(ctx, "mcp:initialize"); err != nil {
        return nil, fmt.Errorf("authorization failed: %w", err)
    }
    
    // Phase 4: Rate limiting
    if err := s.rateLimiter.Allow(ctx, req.Security.IPAddress); err != nil {
        return nil, fmt.Errorf("rate limit exceeded: %w", err)
    }
    
    // Phase 5: Security capability negotiation
    secureCapabilities := s.negotiateSecurityCapabilities(req.Capabilities)
    
    // Success - create session
    session := &MCPSession{
        ID:           generateSessionID(),
        ClientInfo:   req.ClientInfo,
        Principal:    GetPrincipal(ctx),
        Capabilities: secureCapabilities,
        CreatedAt:    time.Now(),
        LastActivity: time.Now(),
    }
    
    s.sessions[session.ID] = session
    
    s.logger.Info("mcp_session_established",
        "session_id", session.ID,
        "client_name", req.ClientInfo.Name,
        "client_version", req.ClientInfo.Version,
        "auth_method", req.Authentication.Method,
        "security_level", s.getSecurityLevel(session))
    
    return &types.InitializeResult{
        ProtocolVersion: s.config.ProtocolVersion,
        Capabilities:    secureCapabilities,
        ServerInfo: types.ServerInfo{
            Name:    "mcpeg",
            Version: s.config.Version,
        },
    }, nil
}
```

### 5. Bar-Raising Security Features

**Beyond Standard MCP:**

1. **Request Signing & Integrity**
```go
type SignedMCPRequest struct {
    types.Request
    Signature string    `json:"signature"`
    Timestamp time.Time `json:"timestamp"`
    Nonce     string    `json:"nonce"`
}

// Verify request integrity
func (s *MCPServer) VerifyRequestIntegrity(req SignedMCPRequest) error {
    // Check timestamp (prevent replay attacks)
    if time.Since(req.Timestamp) > 5*time.Minute {
        return errors.New("request timestamp too old")
    }
    
    // Verify signature
    expectedSig := s.signRequest(req.Request, req.Timestamp, req.Nonce)
    if !hmac.Equal([]byte(req.Signature), []byte(expectedSig)) {
        return errors.New("invalid request signature")
    }
    
    return nil
}
```

2. **Capability-Based Security**
```go
type SecureCapability struct {
    Name         string   `json:"name"`
    RequiredAuth []string `json:"required_auth"`
    AllowedIPs   []string `json:"allowed_ips"`
    TimeWindows  []string `json:"time_windows"`
    RateLimit    int      `json:"rate_limit"`
}

// Fine-grained capability authorization
func (s *MCPServer) AuthorizeCapability(ctx context.Context, capability string) error {
    principal := GetPrincipal(ctx)
    if principal == nil {
        return errors.New("no authenticated principal")
    }
    
    cap, exists := s.capabilities[capability]
    if !exists {
        return errors.New("capability not found")
    }
    
    // Check required authentication level
    if !hasRequiredAuth(principal, cap.RequiredAuth) {
        return errors.New("insufficient authentication level")
    }
    
    // Check IP restrictions
    if !isAllowedIP(GetClientIP(ctx), cap.AllowedIPs) {
        return errors.New("IP not allowed for this capability")
    }
    
    // Check time windows
    if !isWithinTimeWindow(time.Now(), cap.TimeWindows) {
        return errors.New("capability not available at this time")
    }
    
    return nil
}
```

3. **Security Monitoring & Alerting**
```go
type SecurityMonitor struct {
    alerts  chan SecurityAlert
    metrics metrics.Metrics
    logger  logging.Logger
}

type SecurityAlert struct {
    Type        string                 `json:"type"`
    Severity    string                 `json:"severity"`
    Description string                 `json:"description"`
    Context     map[string]interface{} `json:"context"`
    Timestamp   time.Time             `json:"timestamp"`
}

func (sm *SecurityMonitor) DetectAnomalies(ctx context.Context, req types.Request) {
    // Rate limiting violations
    if sm.isRateLimitViolation(ctx) {
        sm.alerts <- SecurityAlert{
            Type:        "rate_limit_violation",
            Severity:    "medium",
            Description: "Client exceeded rate limit",
            Context:     map[string]interface{}{
                "client_ip": GetClientIP(ctx),
                "method": req.Method,
            },
        }
    }
    
    // Suspicious request patterns
    if sm.isSuspiciousPattern(ctx, req) {
        sm.alerts <- SecurityAlert{
            Type:        "suspicious_pattern",
            Severity:    "high",
            Description: "Unusual request pattern detected",
        }
    }
}
```

### 6. Configuration Security

**Production Security Configuration:**
```yaml
security:
  # Authentication
  auth:
    required: true
    methods: ["jwt", "mtls"]
    jwt:
      secret: "${JWT_SECRET}"
      issuer: "mcpeg.service"
      audience: "mcp.clients"
      expiry: "1h"
      
  # Authorization  
  authz:
    rbac_enabled: true
    default_deny: true
    policy_file: "config/rbac_policies.yaml"
    
  # Rate limiting
  rate_limit:
    global:
      requests_per_minute: 1000
      burst: 100
    per_client:
      requests_per_minute: 100
      burst: 10
    per_method:
      "tools/call": 50
      "resources/read": 200
      
  # Request validation
  validation:
    max_request_size_mb: 10
    max_response_size_mb: 50
    sanitize_input: true
    validate_schemas: true
    
  # Monitoring
  monitoring:
    log_all_requests: true
    log_failures: true
    alert_on_anomalies: true
    security_headers: true
    
  # Session management
  sessions:
    timeout: "30m"
    max_concurrent: 100
    secure_cookies: true
    csrf_protection: true
```

## Implementation Priority

**Phase 1: Foundation Security (Week 1)**
- TLS termination with strong ciphers
- Basic authentication (API key/JWT)
- Request validation and sanitization

**Phase 2: Enhanced Security (Week 2)**
- Mutual TLS support
- RBAC authorization
- Rate limiting per client/method

**Phase 3: Advanced Security (Week 3)**
- Request signing and integrity checks
- Security monitoring and alerting
- Anomaly detection

**Phase 4: Bar-Raising Features (Week 4)**
- Capability-based fine-grained authorization
- Advanced threat detection
- Security analytics and reporting

## Consequences

### Positive
- **Production Ready**: Meets enterprise security requirements
- **Defense in Depth**: Multiple security layers
- **LLM Debuggable**: Security events logged with full context
- **Compliance Ready**: Supports audit requirements
- **Future Proof**: Extensible security framework

### Negative
- **Complexity**: Significant security infrastructure
- **Performance**: Security checks add latency
- **Configuration**: Many security knobs to configure

### Neutral
- **Standard Practice**: Industry-standard security patterns
- **Operational Overhead**: Security monitoring and maintenance

## Security Compliance

This design enables compliance with:
- **SOC 2**: Access controls, encryption, monitoring
- **GDPR**: Data protection, audit trails, right to deletion
- **HIPAA**: Encryption, access controls, audit logs
- **PCI DSS**: Strong authentication, encrypted transport

The bar-raising approach ensures MCPEG exceeds basic MCP security requirements!