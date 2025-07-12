# ADR-019: Brand Identity and User Experience Architecture

## Status
**ACCEPTED** - *2025-07-11*

## Context

MCpeg transitioned from a technical prototype to a production-ready gateway service requiring comprehensive brand identity and user experience architecture. The system needed consistent visual identity, clear naming conventions, and professional presentation across all user touchpoints.

Key requirements identified:
- Professional visual identity for enterprise adoption
- Clear, memorable naming that conveys purpose
- Consistent presentation across CLI, API, and documentation
- Scalable design assets for various use cases
- User-friendly pronunciation and recognition

## Decision

We established a comprehensive brand identity and user experience architecture centered around the "MCpeg" brand identity with the following design principles:

### Brand Identity Framework

#### 1. **Name Evolution**: "MC PEG" → "MCpeg"
- **Initial Decision**: "MC PEG" (pronounced "em-see peg")
- **Refinement**: "MCpeg" for cleaner, modern single-word appearance
- **Pronunciation**: Maintained as "MC peg" for natural flow
- **Rationale**: Clean visual presentation while preserving intuitive pronunciation

#### 2. **Logo Design Architecture**
```svg
<!-- Core Design Elements -->
- Hexagonal "peg" shape (represents connector concept)
- Connection lines (shows gateway/bridging functionality)  
- Clean typography with service name
- Professional blue color palette (#2563eb)
- Scalable SVG format for all applications
```

#### 3. **Visual Design System**
- **Primary Color**: Professional blue (#2563eb)
- **Typography**: Clean, geometric fonts
- **Symbol**: Hexagonal peg with connection lines
- **Tagline**: "The Peg That Connects Model Contexts"
- **File Format**: SVG for scalability and crispness

### User Experience Principles

#### 1. **Consistent Brand Application**
```go
// CLI Interface Standardization
fmt.Printf("MCpeg Gateway v%s - The Peg That Connects Model Contexts\n", version)
fmt.Printf("Pronunciation: MC peg (em-see peg)\n")

// API Documentation Headers
"service": "MCpeg Gateway",
"tagline": "The Peg That Connects Model Contexts"
```

#### 2. **Professional Presentation**
- All user-facing interfaces use consistent "MCpeg" branding
- Help text includes pronunciation guide for clarity
- Version information reinforces brand identity
- API responses include service identification

#### 3. **Asset Organization**
```
assets/
└── logo.svg          # Master logo file
    ├── Hexagonal peg shape
    ├── Connection lines
    ├── MCpeg typography
    └── Professional styling
```

## Implementation Details

### Phase 1: Brand Establishment (Commit 1e44727)
```bash
# Created comprehensive brand foundation
- assets/logo.svg: Professional logo with hexagon and connections
- README.md: Updated with logo, pronunciation, and branding
- CLI interfaces: Updated all help text and version info
- API docs: Added service branding to admin endpoints
```

### Phase 2: Brand Refinement (Commit 5c4156c)
```bash
# Refined to modern single-word presentation
- Updated "MC PEG" → "MCpeg" across all interfaces
- Maintained pronunciation guide as "MC peg"
- Enhanced professional appearance
- Consistent application across all touchpoints
```

### Phase 3: Logo Optimization (Commit 1f51dca)
```bash
# Simplified logo for maximum impact
- Removed redundant "GATEWAY" text
- Increased MCpeg text prominence
- Improved visual balance and focus
- Stronger, more memorable brand presence
```

### User Interface Integration
```go
// gateway_server.go - Brand integration
func (s *GatewayServer) setupAdminRoutes() {
    s.adminMux.HandleFunc("/admin/info", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]string{
            "service":     "MCpeg Gateway",
            "version":     s.version,
            "tagline":     "The Peg That Connects Model Contexts",
            "status":      "running",
        })
    })
}
```

## Consequences

### Positive
- **Professional Identity**: Enterprise-ready visual presentation
- **Clear Communication**: "MC peg" pronunciation removes ambiguity
- **Memorable Branding**: Hexagonal peg metaphor reinforces purpose
- **Consistent Experience**: Unified presentation across all interfaces
- **Scalable Assets**: SVG format ensures quality at all sizes
- **Market Positioning**: Professional appearance enhances adoption

### Negative
- **Migration Required**: Existing documentation and integrations need updates
- **Brand Recognition**: New brand requires time to establish recognition
- **Asset Maintenance**: Logo and brand assets require ongoing management

## Technical Implementation

### Files Created/Modified
- `assets/logo.svg`: Master brand asset with hexagonal peg design
- `README.md`: Updated with logo, branding, and pronunciation guide
- `CHANGELOG.md`: Added branding section documenting identity
- `internal/server/gateway_server.go`: Integrated brand into API responses

### Brand Guidelines
1. **Always use "MCpeg"** in user-facing text (single word, capital MC)
2. **Always include pronunciation guide** in help text: "MC peg (em-see peg)"
3. **Use hexagonal logo** for visual brand representation
4. **Apply blue color palette** (#2563eb) for brand consistency
5. **Include tagline** where appropriate: "The Peg That Connects Model Contexts"

### Quality Assurance
- **Visual Consistency**: All interfaces checked for brand compliance
- **Pronunciation Clarity**: Help text verified for pronunciation guidance
- **Asset Quality**: Logo tested at multiple scales and formats
- **User Testing**: CLI and API interfaces validated for brand presentation

## References
- [Brand Assets Directory](../../assets/)
- [Logo Design File](../../assets/logo.svg)
- [CLI User Experience Guide](../guidelines/cli-ux.md)
- [API Documentation Standards](../guidelines/api-docs.md)
- [Visual Identity Guidelines](../guidelines/visual-identity.md)

## Related ADRs
- [ADR-001: Record Architecture Decisions](001-record-architecture-decisions.md)
- [ADR-016: Unified Binary Architecture](016-unified-binary-architecture.md)
- [ADR-018: Production HTTP Middleware Architecture](018-production-http-middleware-architecture.md)