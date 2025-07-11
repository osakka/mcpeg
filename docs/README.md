# Documentation Directory

This directory contains all project documentation.

## Structure

- `/adrs/` - Architecture Decision Records
  - Timeline-based record of all architectural decisions
  - Each ADR follows the standard ADR template
  - Never modify past ADRs; create new ones to supersede
  
- `/guidelines/` - Development and operational guidelines
  - Git hygiene practices
  - Code style guides
  - Contribution guidelines
  - Security guidelines
  
- `/api/` - API documentation (generated)
  - Auto-generated from API schemas
  - Do not edit manually
  
- `/architecture/` - System architecture documentation
  - High-level design documents
  - Component interaction diagrams
  - Deployment architecture

## Documentation Principles

1. **No Redundancy**: Information should exist in exactly one place
2. **Always Current**: Use automation to keep docs synchronized with code
3. **Clarity**: Write for your audience (developers, operators, users)
4. **Versioned**: Documentation versions should match code versions

## Contributing to Documentation

1. For architectural changes, create a new ADR
2. Update guidelines when processes change
3. Ensure all diagrams have source files (e.g., .puml, .drawio)
4. Run documentation validation before committing