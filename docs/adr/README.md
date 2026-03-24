# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the SereniBase project.

## What is an ADR?

An ADR is a document that captures an important architectural decision made along with its context and consequences.

## ADR Index

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-0001](0001-use-golang.md) | Use Go as Primary Language | Accepted |
| [ADR-0002](0002-postgresql-database.md) | Use PostgreSQL as Primary Database | Accepted |
| [ADR-0003](0003-gin-web-framework.md) | Use Gin Web Framework | Accepted |
| [ADR-0004](0004-jwt-authentication.md) | Use JWT for Authentication | Accepted |
| [ADR-0005](0005-rbac-authorization.md) | Implement RBAC for Authorization | Accepted |

## Template

When creating a new ADR, use the following template:

```markdown
# ADR-NNNN: Title

## Status
Proposed | Accepted | Deprecated | Superseded

## Context
What is the issue that we're seeing that is motivating this decision?

## Decision
What is the change that we're proposing and/or doing?

## Consequences
What becomes easier or more difficult to do because of this change?
```
