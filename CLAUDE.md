# CLAUDE.md - AI Assistant Context & Instructions

This file provides essential context for AI assistants working on the MCPEG project.

## Project Overview

MCPEG (Model Context Protocol Enablement Gateway) is a lightweight service that provides a Model Context Protocol (MCP) API on one side and integrates with external services via API calls or binary invocations on the other side.

## Key Context for AI Assistants

### Development Methodology: XVC Framework

This project follows the [XVC (Extreme Vibe Coding)](https://github.com/osakka/xvc) principles for human-LLM collaboration:

1. **Single Source of Truth**: Every piece of information exists in exactly one place
2. **No Redundancy**: Eliminate duplication across all systems
3. **Surgical Precision**: Every change is intentional and well-documented
4. **Bar-Raising Solutions**: Only implement patterns that improve the overall system
5. **Forward Progress Only**: No regression, always building on solid foundations
6. **Always Solve Never Mask**: Address root causes, not symptoms

### Current Architecture

- **Unified Binary**: Single `mcpeg` binary with subcommands (`gateway`, `codegen`, `validate`)
- **API-First**: All functionality derives from OpenAPI specifications
- **Single Source of Truth Build**: All build logic centralized in `scripts/build.sh`
- **Module Path**: `github.com/osakka/mcpeg`

### Essential Patterns

1. **Build System**: Always use `scripts/build.sh` or delegate via Makefile
2. **CLI Interface**: Use `mcpeg <subcommand>` pattern consistently
3. **Logging**: LLM-optimized structured logging for complete debuggability
4. **Error Handling**: Comprehensive error context for troubleshooting
5. **Documentation**: All decisions documented in ADRs

### Standardized "Wrapup" Checklist

When completing any significant task, execute this checklist when the user mentions 'wrapup':

#### 1. Code Quality & Consistency
- [ ] Run linting and formatting tools
- [ ] Verify all imports use correct module path (`github.com/osakka/mcpeg`)
- [ ] Ensure unified binary usage throughout codebase
- [ ] Check for any remaining separate binary references

#### 2. Documentation Updates
- [ ] Create or update relevant ADR if architectural decisions were made
- [ ] Update CHANGELOG.md with changes (Added/Changed/Fixed sections)
- [ ] Verify README.md reflects current functionality
- [ ] Update project structure documentation if needed
- [ ] Check all documentation for module path consistency

#### 3. Testing & Validation
- [ ] Build and test the unified binary functionality
- [ ] Verify all subcommands work correctly
- [ ] Run any existing tests
- [ ] Validate OpenAPI specifications if modified

#### 4. Git Operations
- [ ] Stage all relevant changes
- [ ] Create descriptive commit message following project patterns
- [ ] Include "🤖 Generated with Claude Code" footer
- [ ] Add "Co-Authored-By: Claude" line
- [ ] Push to remote repository (if requested)

#### 5. Communication
- [ ] Summarize what was accomplished
- [ ] Note any breaking changes or migration requirements
- [ ] Highlight any remaining work or follow-up tasks

### Common Patterns to Follow

#### Commit Message Format
```
feat: implement [feature description]

[Detailed explanation of changes]

Changes:
- Bullet point list of specific changes
- Focus on the "what" and "why"

Benefits:
- List of benefits or improvements

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

#### File Structure Principles
- **cmd/**: Application entry points only
- **internal/**: Private application code
- **pkg/**: Public reusable packages
- **api/**: OpenAPI specifications
- **docs/**: All documentation with clear categorization
- **scripts/**: Build and utility scripts
- **build/**: Build artifacts (gitignored)

#### Development Workflow
1. Always read existing code to understand patterns
2. Use existing utilities and frameworks
3. Follow single source of truth principle
4. Document architectural decisions in ADRs
5. Update CHANGELOG.md for all changes
6. Maintain unified binary architecture

### Build System Usage

```bash
# Standard commands
./scripts/build.sh build      # Build unified binary
./scripts/build.sh dev        # Start development server  
./scripts/build.sh test       # Run tests
./scripts/build.sh validate   # Validate OpenAPI specs

# Via Makefile (delegates to build script)
make build
make dev
make test
make validate
```

### Current Status

- ✅ Unified binary architecture implemented
- ✅ Single source of truth build system
- ✅ Comprehensive service registry and routing
- ✅ OpenAPI-based code generation
- 🔄 Service adapter implementations (in progress)
- 📋 Additional discovery mechanisms (planned)

### Important Notes

- **Never create separate binaries** - always use unified `mcpeg` binary
- **Always check module paths** - use `github.com/osakka/mcpeg`
- **Build system is source of truth** - modify `scripts/build.sh` not Makefile
- **Document decisions** - create ADRs for architectural changes
- **LLM-optimized logging** - every log entry should provide complete context

This context file should be consulted before making any significant changes to ensure consistency with project principles and patterns.