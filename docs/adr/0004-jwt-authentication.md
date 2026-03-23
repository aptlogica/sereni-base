# ADR-0004: Use JWT for Authentication

## Status

Accepted

## Context

SereniBase requires an authentication mechanism that:

- Supports stateless authentication for horizontal scaling
- Can include user claims and permissions
- Works well with SPA and mobile clients
- Has industry-standard security practices
- Allows token refresh without re-authentication

Alternatives considered:
- **Session-based auth**: Requires server-side state and session storage
- **OAuth2 only**: Adds complexity for simple use cases
- **API Keys**: Less suitable for user authentication
- **PASETO**: Newer but smaller ecosystem

## Decision

We will use **JWT (JSON Web Tokens)** for authentication in SereniBase.

Implementation details:
- Use RS256 (RSA) or HS256 (HMAC) signing algorithms
- Short-lived access tokens (15 minutes)
- Long-lived refresh tokens (7 days) stored securely
- Include user ID, roles, and tenant information in claims
- Use golang-jwt/jwt/v5 library

## Consequences

### Positive
- Stateless authentication enables horizontal scaling
- Self-contained tokens reduce database lookups
- Standard format understood by many tools
- Easy to implement token refresh flow
- Can be validated without network calls

### Negative
- Tokens cannot be revoked without additional infrastructure
- Token size larger than simple session IDs
- Requires careful secret/key management
- Clock skew can cause validation issues
- Refresh token rotation adds complexity
