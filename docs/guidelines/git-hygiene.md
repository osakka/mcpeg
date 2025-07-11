# Git Hygiene Guidelines

## Commit Messages

### Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, semicolons, etc.)
- `refactor`: Code refactoring without functionality change
- `perf`: Performance improvements
- `test`: Test additions or modifications
- `build`: Build system changes
- `ci`: CI/CD configuration changes
- `chore`: Maintenance tasks

### Example
```
feat(mcp): implement resource listing endpoint

- Add handler for resources/list method
- Include pagination support
- Add comprehensive test coverage

Closes #123
```

## Branch Strategy

### Branch Types
- `main` - Production-ready code
- `develop` - Integration branch
- `feature/*` - New features
- `fix/*` - Bug fixes
- `release/*` - Release preparation

### Branch Naming
- Use descriptive names: `feature/add-yaml-config-validation`
- Include issue numbers: `fix/123-handle-null-resources`

## Pull Request Guidelines

1. **Title**: Clear description of changes
2. **Description**: 
   - What changed and why
   - How to test
   - Breaking changes
3. **Size**: Keep PRs small and focused
4. **Reviews**: Require at least one approval

## Best Practices

1. **Commit Often**: Small, logical commits
2. **Pull Before Push**: Always pull latest changes
3. **No Force Push**: Except on personal branches
4. **Sign Commits**: Use GPG signing when possible
5. **Clean History**: Squash commits when merging
6. **Update Regularly**: Rebase feature branches on develop

## Forbidden Practices

1. Committing secrets or credentials
2. Large binary files (use Git LFS if needed)
3. Commented-out code
4. Console logs in production code
5. Merge commits in feature branches (use rebase)

## Git Hooks

Recommended pre-commit hooks:
- Code formatting
- Linting
- Secret scanning
- Test execution