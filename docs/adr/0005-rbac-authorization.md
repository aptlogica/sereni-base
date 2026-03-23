# ADR-0005: Implement RBAC for Authorization

## Status

Accepted

## Context

SereniBase needs a flexible authorization system that:

- Controls access to resources at a granular level
- Supports multiple tenants with isolated permissions
- Allows custom roles per organization
- Is auditable and easy to understand
- Can scale with growing permission sets

Alternatives considered:
- **ACL (Access Control Lists)**: Complex to manage at scale
- **ABAC (Attribute-Based)**: Flexible but complex to implement
- **ReBAC (Relationship-Based)**: Good for social graphs but overkill
- **Simple role checks**: Too inflexible for enterprise needs

## Decision

We will implement **RBAC (Role-Based Access Control)** for authorization.

Implementation details:
- **Resources**: Entities that can be accessed (tables, workspaces, etc.)
- **Actions**: Operations on resources (create, read, update, delete)
- **Permissions**: Resource + Action combinations
- **Roles**: Named collection of permissions
- **User-Role Assignment**: Per-tenant role assignments

Key tables:
- `permissions`: Defines available permissions
- `roles`: Defines roles with names and descriptions
- `role_permissions`: Maps permissions to roles
- `user_roles`: Maps roles to users with tenant scope

## Consequences

### Positive
- Clear separation of duties
- Easy to audit who can do what
- Roles map to business functions
- Scales well with proper caching
- Easy to understand for administrators

### Negative
- Role explosion if too granular
- Requires careful role design
- Permission changes need role updates
- Some edge cases need additional rules
- Initial setup complexity
