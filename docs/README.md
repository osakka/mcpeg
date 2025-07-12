# MCpeg Documentation

**MCpeg** (Model Context Protocol Enablement Gateway) is a production-ready MCP gateway that bridges MCP-compliant clients with diverse backend services through a comprehensive plugin architecture.

## Quick Navigation

### 🚀 Getting Started
- **[Quick Start Guide](guides/quick-start.md)** - Get MCpeg running in 5 minutes
- **[Installation Guide](guides/installation.md)** - Complete setup instructions
- **[User Guide](guides/user-guide.md)** - Comprehensive user manual

### 📚 Core Documentation

#### Architecture and Design
- **[High-Level Design](architecture/high-level-design.md)** - System architecture overview
- **[Project Structure](architecture/project-structure.md)** - Codebase organization
- **[Architecture Decisions](adrs/README.md)** - Complete ADR timeline (001-027)

#### API and Integration
- **[API Reference](reference/api-reference.md)** - Complete REST API documentation
- **[MCP Protocol Reference](reference/mcp-protocol.md)** - MCP implementation details
- **[Plugin Development](guides/plugin-development.md)** - Creating custom plugins

#### Configuration and Deployment
- **[Configuration Reference](reference/configuration.md)** - All configuration options
- **[Deployment Guide](guides/deployment.md)** - Production deployment
- **[CLI Reference](reference/cli-reference.md)** - Command-line interface

### 🛠️ Development

#### For Developers
- **[Developer Guide](guides/developer-guide.md)** - Development environment setup
- **[XVC Methodology](development/xvc-methodology.md)** - Development framework
- **[Testing Methodology](development/testing-methodology.md)** - Testing approach
- **[Git Hygiene](development/git-hygiene.md)** - Git best practices

#### For Contributors
- **[Contributing Guide](processes/contributing.md)** - How to contribute
- **[Code Review Process](processes/code-review.md)** - Review procedures
- **[Release Process](processes/release-process.md)** - Version management

### 📖 Reference Materials

#### Technical Specifications
- **[Performance Specifications](reference/performance.md)** - Benchmarks and characteristics
- **[Security Model](reference/security-model.md)** - Authentication and authorization
- **[Protocol Compliance](reference/protocol-compliance.md)** - MCP 2025-03-26 compliance

#### Troubleshooting
- **[Troubleshooting Guide](guides/troubleshooting.md)** - Common issues and solutions
- **[Monitoring Guide](guides/monitoring.md)** - Observability setup

## Documentation Organization

MCpeg documentation is organized into logical categories:

| Category | Purpose | Audience |
|----------|---------|----------|
| **[Architecture](architecture/README.md)** | System design and structure | Architects, senior developers |
| **[ADRs](adrs/README.md)** | Architecture decision records | All team members |
| **[Guides](guides/README.md)** | Task-oriented how-to documentation | Users, developers, operators |
| **[Reference](reference/README.md)** | Comprehensive reference materials | Developers, integrators |
| **[Development](development/README.md)** | Development methodology and standards | Contributors, developers |
| **[Processes](processes/README.md)** | Project workflows and governance | Contributors, maintainers |

## Documentation Principles

All MCpeg documentation follows these standards:

1. **100% Factual** - No exaggerations or unverified claims
2. **Single Source of Truth** - Each piece of information exists exactly once
3. **Clear and Crisp** - Concise, actionable content
4. **Consistent** - Uniform structure, naming, and style
5. **Current** - Always matches the latest codebase implementation
6. **Accessible** - Clear navigation and cross-references

## Finding Information

### By Task
- **Setting up MCpeg**: Start with [Quick Start Guide](guides/quick-start.md)
- **Integrating with MCpeg**: See [API Reference](reference/api-reference.md)
- **Developing plugins**: Read [Plugin Development Guide](guides/plugin-development.md)
- **Contributing code**: Follow [Contributing Guide](processes/contributing.md)
- **Understanding decisions**: Browse [Architecture Decisions](adrs/README.md)

### By Role
- **End Users**: [User Guide](guides/user-guide.md) → [Configuration Reference](reference/configuration.md)
- **Developers**: [Developer Guide](guides/developer-guide.md) → [API Reference](reference/api-reference.md)
- **Operators**: [Deployment Guide](guides/deployment.md) → [Monitoring Guide](guides/monitoring.md)
- **Contributors**: [Contributing Guide](processes/contributing.md) → [XVC Methodology](development/xvc-methodology.md)

## Contributing to Documentation

Documentation improvements are welcome! See our [Documentation Standards](processes/documentation-standards.md) for guidelines on:
- Content accuracy requirements
- Style and formatting standards
- Review and validation process
- Cross-reference management

All documentation changes are validated against the current codebase to ensure 100% accuracy.

---

**Version**: Current (matches codebase v1.0.0)  
**Last Updated**: 2025-07-12  
**Maintenance**: [Documentation Standards](processes/documentation-standards.md)