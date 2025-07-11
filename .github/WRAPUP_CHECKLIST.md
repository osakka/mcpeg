# Wrapup Checklist

Execute this checklist whenever a user says "wrapup" after completing a task.

## 📋 Complete Wrapup Process

### 1. Documentation Updates
- [ ] Update `CHANGELOG.md` with new features/changes
- [ ] Create or update relevant ADRs in `docs/adrs/`
- [ ] Update `README.md` with new features and usage examples
- [ ] Update deployment/installation documentation if applicable
- [ ] Update any affected configuration examples

### 2. Code Quality
- [ ] Verify clean build: `make build`
- [ ] Test binary functionality: `./build/mcpeg --version`
- [ ] Check for any broken imports or references
- [ ] Ensure all new files follow project structure

### 3. Version Control
- [ ] Check git status: `git status`
- [ ] Stage all changes: `git add .`
- [ ] Review staged changes: `git diff --staged --stat`
- [ ] Commit with descriptive message following format:
  ```
  type: short description
  
  - Bullet points of key changes
  - Include any breaking changes
  - Reference any ADRs or issues
  
  🤖 Generated with [Claude Code](https://claude.ai/code)
  
  Co-Authored-By: Claude <noreply@anthropic.com>
  ```
- [ ] Push changes: `git push`
- [ ] Tag release if appropriate: `git tag vX.Y.Z && git push --tags`

### 4. Final Verification
- [ ] Verify push succeeded
- [ ] Check that documentation is consistent
- [ ] Confirm binary works as expected
- [ ] Update project status in README if major milestone

## 🎯 Commit Message Types

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding/updating tests
- `chore:` - Build system, dependencies, etc.
- `breaking:` - Breaking changes

## 📝 ADR Creation Guidelines

When creating new ADRs:
- Use format: `ADR-XXX: Title`
- Include: Status, Context, Decision, Consequences
- Reference implementation files
- Document alternatives considered
- Add to ADR index if exists

## ✅ Success Criteria

Wrapup is complete when:
- [ ] All changes are committed and pushed
- [ ] Documentation reflects current state
- [ ] Binary builds and runs correctly
- [ ] No broken references or missing files
- [ ] Future contributors can understand changes from docs alone

---
*This checklist ensures consistent project maintenance following XVC methodology.*