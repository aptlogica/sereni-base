# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Base management API (`/api/v1/{bs}`): create, rename and delete tables; add, change, delete and reorder columns (with optional per-column constraints and type changes with `dry_run`); list, add, change, check and drop unique and check constraints; show and set the primary key. Off by default per base (`settings` jsonb on `base_api`, owners and co-owners only via `PATCH /api/v1/base/{id}/api/settings`); writes run as the user who turned it on, fall back to the base owner, and are serialised per base. Check conditions are structured JSON compiled to SQL on the server.
- Background refresh of the schema snapshot and OpenAPI spec after schema writes, plus a periodic sweep.
- New API error codes: `in_use`, `forbidden`, `feature_disabled`, `conversion_not_allowed`, `system_column`, `type_not_supported`, `unavailable`.
- Base management routes need no secret key yet, like the rest of the base API (open access, dev only); the secret key check and its error message come with the security work.
- Per-base OpenAPI spec (`/docs`, `/redoc`, `/openapi.json`) lists the base management paths only while the feature is on.

## [0.1.0-beta] - 2026-03-24
### Added
- Initial open-source hygiene files added
- Comprehensive A+ readiness audit compliance
- CODEOWNERS file for branch protection
- Complete test infrastructure with 40+ new test cases
- Security validation and dev container configuration

### Changed
- Project status updated to Beta for public testing

### Note
This is an **active beta version**. APIs and features are subject to change. Please report issues and provide feedback.
