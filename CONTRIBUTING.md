# Contributing to SereniBase

Thank you for considering contributing to SereniBase! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Community](#community)

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to support@aptlogica.com.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/sereni-base.git
   cd sereni-base
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/aptlogica/sereni-base.git
   ```
4. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose
- PostgreSQL 15+ (or use Docker)
- Make

### Environment Setup

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

2. Update `.env` with your configuration

3. Start dependencies:
   ```bash
   docker-compose up -d postgres minio
   ```

4. Run the application:
   ```bash
   make run
   ```

5. Run tests:
   ```bash
   make test
   ```

## Making Changes

### Branch Naming Convention

Use descriptive branch names following these patterns:

- `feature/<description>` - New features
- `fix/<description>` - Bug fixes
- `docs/<description>` - Documentation changes
- `refactor/<description>` - Code refactoring
- `test/<description>` - Test additions or changes
- `chore/<description>` - Maintenance tasks

### Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `perf`: Performance improvement
- `test`: Adding or updating tests
- `build`: Build system or external dependencies
- `ci`: CI configuration files
- `chore`: Other changes that don't modify src or test files

**Examples:**
```
feat(auth): add password reset functionality
fix(api): handle nil pointer in user handler
docs(readme): update installation instructions
```

## Pull Request Process

1. **Update your fork** with the latest upstream changes:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Ensure your changes pass all checks**:
   ```bash
   make lint
   make test
   ```

3. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

4. **Create a Pull Request** on GitHub with:
   - Clear title following commit conventions
   - Description of changes
   - Link to related issues
   - Screenshots for UI changes

5. **Address review feedback** promptly

6. **Squash commits** if requested before merge

### PR Checklist

- [ ] Code follows the project's coding standards
- [ ] Tests added/updated for changes
- [ ] Documentation updated if needed
- [ ] All CI checks pass
- [ ] No merge conflicts with main

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting (enforced by CI)
- Run `golangci-lint` before committing
- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for exported functions and types

### Project Structure

```
sereni-base/
├── cmd/server/       # Application entry point
├── internal/         # Private application code
│   ├── app/          # Application setup
│   ├── config/       # Configuration
│   ├── handlers/     # HTTP handlers
│   ├── middleware/   # HTTP middleware
│   ├── models/       # Data models
│   ├── services/     # Business logic
│   └── utils/        # Utilities
├── tests/            # Test files
└── docs/             # Documentation
```

### Error Handling

- Always check and handle errors
- Use custom error types for domain errors
- Include context in error messages
- Log errors with appropriate levels

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/handlers/...
```

### Writing Tests

- Place tests in `tests/` directory mirroring `internal/` structure
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for >80% coverage on new code
- Include both positive and negative test cases

### Test Naming

```go
func TestFunctionName_Scenario_ExpectedBehavior(t *testing.T)
```

## Documentation

- Update README.md for user-facing changes
- Add/update API documentation in `docs/`
- Include code comments for complex logic
- Update CHANGELOG.md for notable changes

### API Documentation

We use Swagger/OpenAPI for API documentation:

```bash
make swagger
```

## Community

### Getting Help

- Open an issue for bugs or feature requests
- Use discussions for questions
- Check existing issues before creating new ones

### Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md
- Release notes
- Project documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to SereniBase!
ainer must approve before merging.
