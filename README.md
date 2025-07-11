# MCPEG - Model Context Protocol Enablement Gateway

MCPEG is a lightweight service that provides a Model Context Protocol (MCP) API on one side and integrates with external services via API calls or binary invocations on the other side.

## Overview

MCPEG acts as a bridge between MCP-compliant clients and various backend services, providing:
- Full MCP protocol compliance
- Flexible service integration via REST APIs or binary calls
- YAML-based configuration
- API-first development methodology
- Generated code from official MCP specifications

## Project Structure

- `/src` - Source code including API schemas and implementation
- `/build` - Build artifacts and compiled code
- `/docs` - Documentation, ADRs, and guidelines
- `/CHANGELOG.md` - Version history and changes
- `/CLAUDE.md` - AI assistant context and instructions

## Development Principles

1. **Single Source of Truth**: All API definitions derive from official MCP specifications
2. **No Redundancy**: Each piece of information exists in exactly one place
3. **API-First**: Define APIs before implementation
4. **Code Generation**: Generate code from schemas to ensure consistency
5. **Up-to-date**: Automated processes to keep documentation and code synchronized

## Getting Started

[To be completed after initial implementation]

## License

[To be determined]