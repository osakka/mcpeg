# MCpeg Development Wrapup Checklist

This checklist ensures complete documentation, version control, and quality assurance after any development task.

## **When to Use**
Execute this checklist whenever you mention "wrapup" after completing development tasks.

## **Wrapup Checklist**

### **1. Documentation Updates**
- [ ] **CHANGELOG.md**: Add changes to the "Unreleased" section with proper categorization
- [ ] **ADRs**: Create new Architecture Decision Records for significant changes
- [ ] **README.md**: Update if functionality, setup, or usage changed
- [ ] **API documentation**: Update if new endpoints or changes to existing APIs
- [ ] **Configuration examples**: Update if new config options added

### **2. Version Control**
- [ ] **Git status**: Check for uncommitted changes
- [ ] **Stage changes**: `git add` all relevant files
- [ ] **Commit**: Create descriptive commit message with Claude Code attribution
- [ ] **Push**: Push changes to remote repository

### **3. Quality Assurance**
- [ ] **Build verification**: Ensure project builds successfully
- [ ] **Test execution**: Run relevant tests to verify functionality
- [ ] **Lint/format**: Run linting and formatting tools if available
- [ ] **Configuration validation**: Test with different config files if changed

### **4. ADR Creation Guidelines**
Create ADRs for:
- New architectural patterns or frameworks
- Security implementations
- Configuration changes affecting deployment
- Testing strategy changes
- API modifications
- Infrastructure or process changes

### **5. CHANGELOG Categories**
Use these categories in CHANGELOG.md:
- **Added**: New features, endpoints, capabilities
- **Changed**: Modifications to existing functionality  
- **Deprecated**: Features marked for removal
- **Removed**: Deleted features or code
- **Fixed**: Bug fixes and issue resolutions
- **Security**: Security-related changes

### **6. Commit Message Format**
```
Brief description of change (50 chars max)

- Specific change 1
- Specific change 2  
- Specific change 3

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### **7. Standard ADR Template**
```markdown
# ADR-XXX: [Title]

## Status
**ACCEPTED** - *YYYY-MM-DD*

## Context
[Problem or opportunity description]

## Decision
[What was decided and why]

## Implementation Details
[Technical specifics, code changes, configuration]

## Consequences
### Positive
- [Benefits and improvements]

### Negative  
- [Trade-offs and limitations]

## Files Modified
[List of changed files with brief description]

## Testing
[How the change was verified]

## References
[Links to related ADRs, docs, or external resources]
```

## **Execution Commands**

### Basic Wrapup
```bash
# 1. Check status
git status

# 2. Update documentation
# (Manual: Update CHANGELOG.md, create ADRs, update README.md)

# 3. Stage and commit
git add -A
git commit -m "Your commit message here

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"

# 4. Push changes  
git push origin main

# 5. Verify build
make build

# 6. Run tests
make test  # or go test ./...
```

### Quality Assurance Commands
```bash
# Build verification
make build || echo "Build failed - fix before proceeding"

# Test execution  
go test ./... -timeout 30s

# Format check (if available)
go fmt ./...

# Lint check (if available) 
golangci-lint run || echo "Linting issues found"
```

## **When to Skip Steps**
- **Documentation**: Skip if changes are purely internal refactoring
- **ADRs**: Skip for minor bug fixes or documentation-only changes
- **README**: Skip if no user-facing changes
- **CHANGELOG**: Never skip - always document changes

## **Automation Integration**
This checklist can be automated with:
- Git hooks for pre-commit validation
- CI/CD pipelines for build/test verification  
- Automated changelog generation
- ADR template creation scripts

---

**Last Updated**: 2025-07-11
**Version**: 1.0
**Maintained By**: MCpeg Development Team