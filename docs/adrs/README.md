# Architecture Decision Records

This directory contains all Architecture Decision Records (ADRs) for the MCPEG project.

## What is an ADR?

An Architecture Decision Record captures an important architectural decision made along with its context and consequences.

## ADR Timeline

| Date | ADR | Title | Status |
|------|-----|-------|--------|
| 2025-07-11 | [ADR-001](001-record-architecture-decisions.md) | Record Architecture Decisions | Accepted |
| 2025-07-11 | [ADR-002](002-use-mcp-protocol.md) | Use Model Context Protocol as Core Protocol | Accepted |
| 2025-07-11 | [ADR-003](003-api-first-development.md) | Adopt API-First Development Methodology | Accepted |
| 2025-07-11 | [ADR-004](004-yaml-configuration.md) | Use YAML for Service Configuration | Accepted |
| 2025-07-11 | [ADR-005](005-use-go-language.md) | Use Go as Implementation Language | Accepted |
| 2025-07-11 | [ADR-006](006-prioritize-rest-adapters.md) | Prioritize REST API Adapters | Accepted |
| 2025-07-11 | [ADR-007](007-built-in-validation-framework.md) | Built-in Validation and Testing Framework | Accepted |
| 2025-07-11 | [ADR-008](008-llm-optimized-logging.md) | LLM-Optimized Logging for 100% Troubleshooting | Accepted |
| 2025-07-11 | [ADR-009](009-concurrency-and-memory-management.md) | Concurrency and Memory Management Patterns | Accepted |
| 2025-07-11 | [ADR-010](010-multi-service-gateway.md) | Multi-Service Gateway Architecture | Accepted |
| 2025-07-11 | [ADR-011](011-data-storage-strategy.md) | Data Storage Strategy | Accepted |
| 2025-07-11 | [ADR-012](012-advanced-mcp-services.md) | Advanced MCP Services (Git, Editor, Search, Memory) | Accepted |
| 2025-07-11 | [ADR-013](013-metrics-as-core-infrastructure.md) | Metrics as Core Infrastructure | Accepted |
| 2025-07-11 | [ADR-014](014-missing-infrastructure-components.md) | Missing Infrastructure Components | Accepted |
| 2025-07-11 | [ADR-015](015-mcp-security-and-registration.md) | MCP Security and Registration Requirements | Accepted |
| 2025-07-11 | [ADR-016](016-unified-binary-architecture.md) | Unified Binary Architecture | Accepted |
| 2025-07-11 | [ADR-017](017-production-ready-implementation-complete.md) | Production-Ready Implementation Complete | Accepted |
| 2025-07-11 | [ADR-018](018-production-http-middleware-architecture.md) | Production HTTP Middleware Architecture | Accepted |
| 2025-07-11 | [ADR-019](019-brand-identity-and-user-experience-architecture.md) | Brand Identity and User Experience Architecture | Accepted |
| 2025-07-11 | [ADR-020](020-plugin-system-foundation-architecture.md) | Plugin System Foundation Architecture | Accepted |
| 2025-07-11 | [ADR-021](021-daemon-process-management.md) | Daemon Process Management Architecture | Accepted |
| 2025-07-12 | [ADR-022](022-plugin-registration-service-registry.md) | Plugin Registration Service Registry | Accepted |
| 2025-07-12 | [ADR-023](023-mcp-plugin-integration-phase-1.md) | MCP Plugin Integration Phase 1 | Accepted |
| 2025-07-12 | [ADR-024](024-mcp-plugin-integration-complete-phases-1-4.md) | MCP Plugin Integration Complete (Phases 1-4) | Accepted |
| 2025-07-12 | [ADR-025](025-phase-2-advanced-plugin-discovery-intelligence.md) | Phase 2 Advanced Plugin Discovery and Intelligence | Accepted |
| 2025-07-12 | [ADR-026](026-path-flag-standardization.md) | Path and Flag Standardization | Accepted |
| 2025-07-12 | [ADR-027](027-mcp-resources-read-protocol-compliance.md) | MCP Resources/Read Protocol Compliance | Accepted |

## ADR Status

- **Proposed**: Under discussion
- **Accepted**: Decision approved and in effect
- **Deprecated**: No longer relevant
- **Superseded**: Replaced by another ADR

## Creating a New ADR

1. Copy the [template](template.md)
2. Name it `XXX-short-description.md` (e.g., `005-use-typescript.md`)
3. Fill out all sections
4. Update this README with the new entry
5. Submit for review

## ADR Format

Each ADR follows the format defined in [template.md](template.md).