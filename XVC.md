# XVC Methodology in MCPEG

This document explains how MCPEG implements the [XVC (Extreme Vibe Coding)](https://github.com/osakka/xvc) framework.

## What is XVC?

XVC (Extreme Vibe Coding) is a methodology for effective human-LLM collaboration in software development. It treats LLMs as "pattern reflection engines" and emphasizes establishing consistent interaction patterns for maximum productivity.

## XVC Principles in MCPEG

### 1. Single Source of Truth
- **Implementation**: All API schemas generated from MCP specifications
- **Example**: The `src/api/` directory contains only generated code
- **Benefit**: Zero drift between spec and implementation

### 2. Surgical Precision
- **Implementation**: Every code change is intentional and logged
- **Example**: Our logging framework captures complete context for every operation
- **Benefit**: No accidental complexity

### 3. Bar-Raising Solutions
- **Implementation**: Only patterns that improve the system are adopted
- **Example**: LLM-optimized logging that enables 100% troubleshooting
- **Benefit**: Continuous improvement without technical debt

### 4. Forward Progress Only
- **Implementation**: No breaking changes without migration paths
- **Example**: ADRs document all decisions with clear upgrade paths
- **Benefit**: Stable foundation for rapid development

### 5. Always Solve Never Mask
- **Implementation**: Root cause analysis built into the system
- **Example**: Circuit breakers log failure patterns and suggest fixes
- **Benefit**: Problems are solved once, correctly

## XVC Development Phases in MCPEG

### Phase 1: Initial Learning (Current)
- Establishing patterns with LLM collaboration
- Building foundation utilities (logging, concurrency)
- Documenting all decisions in ADRs

### Phase 2: Productivity (Upcoming)
- Rapid feature development using established patterns
- Implementing MCP protocol handlers
- Building adapter framework

### Phase 3: Proficiency (Future)
- Advanced features with minimal iteration
- Community contributions following XVC patterns
- Ecosystem development

## XVC Practices

### 1. LLM as Pattern Reflector
```go
// Bad: Asking LLM to "figure out" the solution
// "How should I handle errors here?"

// Good: Providing pattern for reflection
// "Apply our error handling pattern with circuit breaker to this REST adapter"
```

### 2. Comprehensive Context
Every component provides full context for LLM understanding:
```go
logger.Error("operation_failed",
    "error_type", "timeout",
    "endpoint", endpoint,
    "duration_ms", duration,
    "suggested_fixes", []string{
        "Increase timeout",
        "Check endpoint health",
        "Enable circuit breaker",
    })
```

### 3. Decision Documentation
All architectural decisions follow ADR template:
- Context (why we need to decide)
- Decision (what we're doing)
- Consequences (what happens as a result)

### 4. Test-Driven Validation
Built-in validation framework ensures correctness:
```go
// Every component includes self-validation
func (adapter *RESTAdapter) Validate() error {
    // Comprehensive validation logic
}
```

## Benefits of XVC in MCPEG

1. **Rapid Development**: Established patterns enable fast feature addition
2. **High Quality**: Bar-raising solutions prevent technical debt
3. **100% Debuggability**: LLMs can troubleshoot any issue from logs
4. **Knowledge Transfer**: Patterns are documented and reusable
5. **Sustainable Pace**: No burnout from fighting bad patterns

## How to Contribute Using XVC

1. **Understand the Pattern**: Read existing code and ADRs
2. **Reflect Don't Invent**: Use established patterns
3. **Document Decisions**: Create ADRs for new patterns
4. **Log Everything**: Enable LLM troubleshooting
5. **Solve Root Causes**: Never work around problems

## XVC Checklist for PRs

- [ ] Follows established patterns
- [ ] Maintains single source of truth
- [ ] Includes comprehensive logging
- [ ] Documents any new patterns in ADRs
- [ ] Solves root cause, not symptoms
- [ ] Includes tests with validation
- [ ] Forward compatible (no breaking changes)

## Resources

- [XVC Framework](https://github.com/osakka/xvc)
- [MCPEG ADRs](/docs/adrs/)
- [LLM Logging Guidelines](/docs/guidelines/llm-logging.md)