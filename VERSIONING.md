# Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/).

## Version Format

```
MAJOR.MINOR.PATCH
```

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality in a backwards-compatible manner
- **PATCH**: Backwards-compatible bug fixes

## Pre-release Versions

Pre-release versions may be denoted by appending a hyphen and identifiers:

```
1.0.0-alpha.1
1.0.0-beta.2
1.0.0-rc.1
```

## Build Metadata

Build metadata may be appended with a plus sign:

```
1.0.0+20260323
1.0.0-beta+exp.sha.5114f85
```

## Version History

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.

## API Stability

### Stable APIs (v1.0.0+)

Once version 1.0.0 is released:

- Public APIs will not have breaking changes within the same major version
- Deprecation notices will be provided at least one minor version before removal
- Security fixes may require breaking changes

### Pre-1.0 APIs

Before version 1.0.0:

- APIs may change between minor versions
- We aim to minimize breaking changes
- Migration guides will be provided for significant changes

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create a git tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push the tag: `git push origin v1.0.0`
5. GitHub Actions will create the release

## Long-Term Support

Major versions will receive:

- **Active Support**: 12 months of feature updates
- **Security Support**: 24 months of security fixes

## Questions

For questions about versioning, please open an issue or contact the maintainers.
