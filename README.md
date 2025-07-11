# MCPEG - Model Context Protocol Enablement Gateway

> ⚠️ **EXPERIMENTAL SOFTWARE**: This project is under heavy development and follows the [XVC (Extreme Vibe Coding)](https://github.com/osakka/xvc) framework for human-LLM collaborative development.

MCPEG is a lightweight service that provides a Model Context Protocol (MCP) API on one side and integrates with external services via API calls or binary invocations on the other side.

## Overview

MCPEG acts as a bridge between MCP-compliant clients and various backend services, providing:
- Full MCP protocol compliance
- Flexible service integration via REST APIs or binary calls
- YAML-based configuration
- API-first development methodology
- Generated code from official MCP specifications

## Project Structure

See [Project Structure Guide](docs/architecture/project-structure.md) for detailed layout.

- `/cmd` - Application entry points
- `/internal` - Private application code  
- `/pkg` - Public Go packages
- `/src` - Generated API schemas only
- `/build` - Build artifacts
- `/docs` - All documentation
  - `/adrs` - Architecture Decision Records
  - `/architecture` - System design documents
  - `/development` - Development guides (XVC, structure, etc.)
  - `/guidelines` - Coding and process guidelines

## Development Methodology: XVC Framework

This project follows the [XVC (Extreme Vibe Coding)](https://github.com/osakka/xvc) principles for human-LLM collaboration. See our [XVC Methodology Guide](docs/guidelines/xvc-methodology.md) for details.

### Core XVC Principles Applied:

1. **Single Source of Truth**: All API definitions derive from official MCP specifications
2. **No Redundancy**: Each piece of information exists in exactly one place  
3. **Surgical Precision**: Every change is intentional and well-documented
4. **Bar-Raising Solutions**: Only implement patterns that improve the overall system
5. **Forward Progress Only**: No regression, always building on solid foundations
6. **Always Solve Never Mask**: Address root causes, not symptoms

### Additional Development Principles:

- **API-First**: Define APIs before implementation
- **Code Generation**: Generate code from schemas to ensure consistency
- **LLM-Optimized Logging**: Every log entry contains complete context for troubleshooting
- **100% Observability**: An LLM can understand system state from logs alone

## Project Status

🚧 **Current Phase**: Foundation Building (Phase 1)

This project is in active development following the XVC methodology phases:
- ✅ Initial pattern establishment with LLM collaboration
- 🔄 Building core infrastructure with bar-raising patterns
- 📋 All decisions documented in ADRs
- 🔍 100% LLM-debuggable through comprehensive logging

## Getting Started

> **Note**: This software is experimental. APIs and functionality may change significantly.

[To be completed after initial implementation]

## Contributing

This project uses XVC methodology. When contributing:
1. Ensure changes align with XVC principles
2. Maintain single source of truth
3. Document decisions in ADRs
4. Write LLM-optimized logs
5. Never mask problems - solve root causes

## License

[To be determined]