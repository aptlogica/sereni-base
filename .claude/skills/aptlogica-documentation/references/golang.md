# Go — GoDoc

Read this with SKILL.md's four things. This file is only the syntax and the
Go-specific traps.

## Style

A doc comment sits immediately above the declaration with no blank line, and
starts with the name of the thing:

```go
// ListForOrg returns one page of an org's leads, newest first.
func (r *LeadRepository) ListForOrg(...) (Page[Lead], error) {}
```

Starting with the name is not a stylistic preference — `go doc`, `pkgsite` and
every editor render the comment as a sentence beginning with that identifier,
so "Returns a page of leads" reads as a fragment and "ListForOrg returns..."
reads as a sentence. `golint`-style checks in `revive` flag the first form.

Use `//` line comments, not `/* */`. Indent code examples by one tab inside the
comment so GoDoc renders them as code.

## Package comment

One file per package carries it — `doc.go` in a package of any size, otherwise
the file named after the package. It names the layer and the entry point:

```go
// Package leads is the lead repository. Repository layer: it owns no
// transaction, returns domain types, and never leaks pgx rows.
//
// Called from: internal/service/leads, internal/agent/outbound.
package leads
```

Every other file in the package gets no package comment — duplicates are a
vet-level annoyance and only one of them is rendered.

## Exported identifiers

Everything exported gets a comment: types, functions, methods, constants,
variables, and struct fields whose name is not self-explanatory.

```go
// LeadRepository reads and writes leads for one org.
//
// It holds a pgx connection bound to one transaction, so construct one per
// request or per graph node. A shared value leaks a transaction between
// requests.
type LeadRepository struct {
    db *pgxpool.Pool

    // orgID scopes every query in this repository. Zero value is never valid;
    // NewLeadRepository rejects it.
    orgID uuid.UUID
}
```

Group related constants under one comment rather than repeating yourself:

```go
// Lifecycle states a node may emit. The set is closed — adding an eighth means
// updating every consumer.
const (
    StateIdle          = "idle"
    StatePlanning      = "planning"
    ...
)
```

## Errors

Sentinel errors and error types are the part of a Go API most likely to be
misused, so document what the caller is expected to do:

```go
// ErrInvalidCursor is returned when a pagination cursor does not decode.
// Callers should treat it as a 400, not a retryable failure.
var ErrInvalidCursor = errors.New("leads: invalid cursor")
```

A function that wraps errors says what it wraps with, so `errors.Is` users know
what they can match:

```go
// ListForOrg returns one page of an org's leads, newest first.
//
// Cursor-based: offset pagination over this table skips rows while writes land.
// Wraps ErrInvalidCursor when the cursor does not decode, and the pgx error
// otherwise.
//
// Called from: internal/transport/leads.List, internal/agent/outbound.Select.
```

## Concurrency

If a type is safe for concurrent use, say so; if it is not, say that too.
Silence here is how a data race ships.

```go
// Registry is safe for concurrent use by multiple goroutines.
// Dispatcher is not: confine one to a single graph run.
```

Document which goroutine owns a channel and who closes it — the type says
neither:

```go
// Events returns the node's lifecycle event stream. The dispatcher owns the
// channel and closes it when the run ends; callers must drain it or leak the
// producing goroutine.
```

## Context and cancellation

Say what a cancelled context does to work already started, especially when
anything external was sent:

```go
// Send publishes the drafted reply. Cancelling ctx after the SMTP handoff does
// not unsend it; the approval record is written before the handoff for exactly
// this reason.
```

## Agent nodes

Same three extra lines as every stack, mirroring the declared values:

```go
// Package draft builds an outbound reply from retrieved context. Agent node.
//
// Tools:   read_crm, search_docs            (mirrors AllowedTools)
// Budget:  40_000 tokens, ceiling 120_000   (mirrors TokenBudget, ContextCeiling)
// Emits:   planning → calling_tool → done, or blocked when context is thin
//
// Called from: internal/agent/outbound.Graph.
//
// Implements: #186
package draft
```

## Unexported code

Unexported identifiers need a comment only when the name cannot carry the
meaning — a non-obvious algorithm, a magic constant, a workaround. Comment the
*why*; the *what* is the code.

```go
// 1 << 14 matches the pgx default batch size; larger batches round-trip fine
// but blow the statement cache.
const batchSize = 1 << 14
```

## Tests

Test names are the documentation: `TestListForOrg_CrossOrgReturnsNotFound`.
A table-driven test documents the table, not each case:

```go
// Each case pins one boundary of the cursor codec. Add a row rather than a new
// test function.
tests := []struct{ ... }{}
```
