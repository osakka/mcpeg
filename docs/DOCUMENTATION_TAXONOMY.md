# MCpeg Documentation Taxonomy

## Overview
This document defines the complete taxonomy, naming schema, and organizational structure for all MCpeg project documentation.

## Documentation Principles
1. **100% Factual**: No exaggerations or unverified claims
2. **Single Source of Truth**: Each piece of information exists in exactly one location
3. **Clear and Crisp**: Concise, actionable content
4. **Consistent**: Uniform structure, naming, and style
5. **Bar-Raising Solutions**: Industry-standard best practices only
6. **Accurate to Implementation**: Documentation matches actual codebase

## Directory Structure

```
docs/
├── README.md                    # Master documentation index
├── architecture/                # System design and structure
│   ├── README.md               # Architecture documentation index
│   ├── high-level-design.md    # System architecture overview
│   ├── project-structure.md    # Codebase organization
│   ├── mcp-endpoint-landscape.md # MCP protocol implementation
│   ├── testing-infrastructure.md # Testing architecture
│   └── complete-architectural-timeline.md # Development timeline
├── adrs/                       # Architecture Decision Records
│   ├── README.md               # ADR index and timeline
│   ├── template.md             # ADR creation template
│   └── NNN-decision-title.md   # Individual ADRs (001-027)
├── guides/                     # User and developer guides
│   ├── README.md               # Guides index
│   ├── user-guide.md           # End-user documentation
│   ├── developer-guide.md      # Development setup and workflows
│   ├── api-guide.md            # API usage documentation
│   └── deployment-guide.md     # Production deployment
├── reference/                  # Reference materials
│   ├── README.md               # Reference index
│   ├── configuration.md        # Configuration reference
│   ├── cli-reference.md        # Command-line interface
│   ├── api-reference.md        # Complete API documentation
│   └── troubleshooting.md      # Problem resolution
├── development/                # Development methodology and standards
│   ├── README.md               # Development documentation index
│   ├── xvc-methodology.md      # XVC framework guidelines
│   ├── git-hygiene.md          # Git best practices
│   ├── coding-standards.md     # Code quality standards
│   └── testing-guidelines.md   # Testing methodology
└── processes/                  # Project processes and workflows
    ├── README.md               # Process documentation index
    ├── contributing.md         # Contribution guidelines
    ├── release-process.md      # Release management
    └── issue-management.md     # Bug reporting and tracking
```

## File Naming Schema

### General Principles
- Use `kebab-case` for all file names
- Descriptive, unambiguous names
- Consistent suffixes for file types
- Maximum 50 characters for readability

### Naming Patterns
- **Architecture**: `{component}-{aspect}.md` (e.g., `high-level-design.md`)
- **ADRs**: `NNN-{decision-summary}.md` (e.g., `001-record-architecture-decisions.md`)
- **Guides**: `{audience}-guide.md` (e.g., `user-guide.md`, `developer-guide.md`)
- **Reference**: `{topic}-reference.md` (e.g., `api-reference.md`, `cli-reference.md`)
- **Process**: `{process-name}.md` (e.g., `contributing.md`, `release-process.md`)

## Content Categories

### 1. Architecture (`architecture/`)
**Purpose**: System design, structure, and technical implementation details
**Audience**: Technical stakeholders, architects, senior developers
**Content**: Design decisions, system diagrams, component interactions

### 2. Architecture Decision Records (`adrs/`)
**Purpose**: Historical record of architectural decisions
**Audience**: All team members, future maintainers
**Content**: Context, decisions, consequences of architectural choices

### 3. User Guides (`guides/`)
**Purpose**: How-to documentation for different audiences
**Audience**: End users, developers, operators
**Content**: Step-by-step instructions, tutorials, workflows

### 4. Reference Materials (`reference/`)
**Purpose**: Comprehensive reference information
**Audience**: Developers, operators, integrators
**Content**: API specs, configuration options, CLI commands

### 5. Development (`development/`)
**Purpose**: Development methodology and standards
**Audience**: Developers, contributors
**Content**: Coding standards, methodologies, best practices

### 6. Processes (`processes/`)
**Purpose**: Project management and workflow documentation
**Audience**: Contributors, maintainers, stakeholders
**Content**: Contribution workflows, release processes, governance

## README.md Structure

### Root Documentation Index (`docs/README.md`)
```markdown
# MCpeg Documentation

## Quick Start
- [User Guide](guides/user-guide.md)
- [API Guide](guides/api-guide.md)

## Architecture
- [High-Level Design](architecture/high-level-design.md)
- [Architecture Decisions](adrs/README.md)

## Reference
- [API Reference](reference/api-reference.md)
- [Configuration](reference/configuration.md)

## Development
- [Developer Guide](guides/developer-guide.md)
- [XVC Methodology](development/xvc-methodology.md)
```

### Category READMEs
Each category directory contains a README.md that:
- Lists all documents in the category
- Provides brief descriptions
- Links to related categories
- Explains the category's purpose

## Content Standards

### Factual Accuracy Requirements
1. **Verify Against Code**: All technical claims verified against actual implementation
2. **Version Alignment**: Documentation matches current codebase version
3. **No Speculation**: Only document what actually exists
4. **Concrete Examples**: Use real, working examples from the codebase

### Style Guidelines
1. **Clear Headers**: Descriptive, action-oriented headings
2. **Concise Language**: Direct, professional tone
3. **Consistent Terminology**: Use project glossary terms
4. **Active Voice**: "Configure the service" not "The service can be configured"

### Cross-Reference Standards
1. **Relative Paths**: Always use relative paths for internal links
2. **Link Validation**: All internal links must be verified
3. **Bidirectional References**: Related documents link to each other
4. **No Orphaned Content**: Every document accessible from main index

## Quality Assurance Process

### Pre-Commit Validation
1. **Accuracy Check**: Technical content verified against implementation
2. **Link Validation**: All internal and external links tested
3. **Style Consistency**: Writing style matches guidelines
4. **Category Placement**: Document in correct taxonomy location

### Regular Maintenance
1. **Monthly Reviews**: Quarterly accuracy audits
2. **Version Synchronization**: Documentation updated with code changes
3. **Link Health**: Automated link checking
4. **User Feedback**: Address documentation issues promptly

This taxonomy ensures MCpeg documentation maintains the highest standards of accuracy, organization, and usability while serving as the definitive single source of truth for all project information.