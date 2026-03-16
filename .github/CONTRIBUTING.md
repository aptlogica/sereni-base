# Contributing to Sereni Base

We love your input! We want to make contributing to this project as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## Pull Request Process

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Testing

We use Go's built-in testing framework. Run tests with:

```bash
make test
# or
go test ./...
```

## Code Coverage

Maintain or improve code coverage with your changes:

```bash
make coverage
# or
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Coding Style

- Use `gofmt` to format your code
- Run `golangci-lint` for linting
- Follow Go idioms and best practices
- Write clear, self-documenting code
- Add comments for exported functions and types

## Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line

Example:
```
Add JWT refresh token rotation

- Implement automatic token rotation
- Add configurable rotation interval
- Update middleware to handle rotation

Fixes #123
```

## Versioning

We use [Semantic Versioning](http://semver.org/). For the versions available, see the [tags on this repository](https://github.com/aptlogica/sereni-base/tags).

## Bug Reports

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/aptlogica/sereni-base/issues).

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## License

By contributing, you agree that your contributions will be licensed under its MIT License.