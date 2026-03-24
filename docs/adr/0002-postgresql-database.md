# ADR-0002: Use PostgreSQL as Primary Database

## Status

Accepted

## Context

SereniBase requires a reliable, scalable database that can handle:

- Complex relational data with multiple table relationships
- ACID transactions for data integrity
- JSON/JSONB storage for flexible schema requirements
- Full-text search capabilities
- Multi-tenant data isolation

Alternatives considered:
- **MySQL/MariaDB**: Good performance but weaker JSON support and fewer advanced features
- **MongoDB**: Flexible schema but lacks ACID transactions and complex joins
- **SQLite**: Simple but not suitable for multi-user concurrent access
- **CockroachDB**: Distributed but adds operational complexity

## Decision

We will use **PostgreSQL** as the primary database for SereniBase.

PostgreSQL provides:
- Robust ACID compliance and transaction support
- Excellent JSONB support for flexible data structures
- Row-level security for multi-tenant isolation
- Advanced indexing (B-tree, GIN, GiST, BRIN)
- Full-text search with configurable language support
- Mature replication and high availability options
- Active community and long-term support

## Consequences

### Positive
- Strong data integrity guarantees
- Flexible querying with both relational and JSON data
- Built-in row-level security for tenant isolation
- Extensive ecosystem of tools and extensions
- Well-understood operational practices

### Negative
- Requires more setup than embedded databases
- Horizontal scaling requires additional tooling (Citus, etc.)
- Connection pooling needed for high concurrency (PgBouncer)
- Schema migrations need careful planning
