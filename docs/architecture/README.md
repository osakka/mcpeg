# MCPEG Architecture Documentation

## Overview

MCPEG implements a gateway pattern to bridge MCP clients with various backend services.

## Contents

- [Project Structure](project-structure.md) - Go project layout and directory organization
- [High-Level Design](high-level-design.md) - System architecture and design patterns
- Component diagrams (coming soon)
- Deployment architecture (coming soon)

## Architecture Diagram

```
┌─────────────────┐     ┌─────────────────────────────────────────────┐
│   MCP Client    │     │                  MCPEG                      │
│ (Claude, etc.)  │     │                                             │
└────────┬────────┘     │  ┌─────────────┐    ┌──────────────────┐  │
         │              │  │ MCP Server  │    │  Configuration   │  │
         │              │  │             │    │  Manager         │  │
         │ JSON-RPC     │  │ - Protocol  │◄───┤                  │  │
         ├─────────────►│  │   Handler   │    │  - YAML Parser   │  │
         │              │  │ - Transport │    │  - Validator     │  │
         │              │  │   Layer     │    │  - Hot Reload    │  │
         │              │  └──────┬──────┘    └──────────────────┘  │
         │              │         │                                   │
         │              │  ┌──────▼──────────────────────┐           │
         │              │  │     Adapter Manager         │           │
         │              │  │                             │           │
         │              │  │  ┌────────┐  ┌────────┐   │           │
         │              │  │  │  REST  │  │ Binary │   │           │
         │              │  │  │Adapter │  │Adapter │   │           │
         │              │  │  └───┬────┘  └────┬───┘   │           │
         │              │  └──────┼────────────┼───────┘           │
         │              │         │            │                     │
         │              │  ┌──────▼────────────▼───────┐           │
         │              │  │  Validation Framework     │           │
         │              │  │                           │           │
         │              │  │  - Config Validator       │           │
         │              │  │  - MCP Compliance Tests   │           │
         │              │  │  - Diagnostic Endpoints   │           │
         │              │  └───────────────────────────┘           │
         │              └─────────────────────────────────────────────┘
         │                              │            │
         │                              ▼            ▼
         │                      ┌──────────┐  ┌──────────┐
         │                      │   REST   │  │  Binary  │
         │                      │   APIs   │  │   Cmds   │
         │                      └──────────┘  └──────────┘
```

## Component Descriptions

### MCP Server
- Implements full MCP protocol specification
- Handles JSON-RPC 2.0 messaging
- Supports multiple transport layers (stdio, HTTP)
- Routes requests to appropriate handlers

### Configuration Manager
- Loads and validates YAML configuration
- Supports hot-reloading of configuration
- Manages service definitions and mappings
- Handles environment variable substitution

### Adapter Manager
- Routes MCP requests to appropriate adapters
- Manages adapter lifecycle
- Handles adapter registration and discovery
- Provides common adapter interfaces

### Adapters
- **REST Adapter**: Translates MCP calls to REST API requests
- **Binary Adapter**: Executes local binaries with sandboxing
- Future: gRPC, GraphQL, Message Queue adapters

### Validation Framework
- Built-in testing capabilities
- Configuration validation
- MCP protocol compliance testing
- Health checks and diagnostics

## Data Flow

1. MCP Client sends JSON-RPC request
2. MCP Server receives and validates request
3. Request routed to appropriate handler
4. Handler consults configuration for mapping
5. Adapter Manager selects correct adapter
6. Adapter translates and executes backend call
7. Response transformed back to MCP format
8. MCP Server sends response to client

## Security Boundaries

- Input validation at MCP layer
- Authentication/authorization per adapter
- Sandboxed execution for binary adapters
- Rate limiting at transport layer
- Audit logging for all operations