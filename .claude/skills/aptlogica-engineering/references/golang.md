# Go — services, pgx, and agent tooling

Read this when working in a Go service. Universal rules are in SKILL.md §1–§5;
this file is how to satisfy them here.

## Toolchain

```bash
gofmt -l .                    # must print nothing
go vet ./...
staticcheck ./...
golangci-lint run             # errcheck, ineffassign, gosec, revive
go test ./... -race -cover
govulncheck ./...             # dependency vulnerabilities, reachability-aware
```

`-race` on every test run. A data race that only manifests under load is the
hardest class of bug to reproduce after the fact, and the race detector finds it
for free in CI.

`govulncheck` is preferred over generic SCA for Go because it does reachability
analysis — it reports the CVE only if your code actually calls the vulnerable
symbol. That materially reduces the noise that trains people to ignore scan
results. See `security-scanning.md`.

`.golangci.yml` baseline:

```yaml
linters:
  enable:
    - errcheck      # unchecked errors — the whole point
    - govet
    - staticcheck
    - ineffassign
    - gosec
    - revive
    - bodyclose     # unclosed HTTP response bodies leak connections
    - contextcheck  # functions that should take a context but don't
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

`errcheck` is the load-bearing one. Go makes discarding an error a single
character, and a discarded error in an agent runtime is indistinguishable from
success to everything downstream.

## Configuration

Parsed and validated at startup so the process refuses to boot on a missing
variable rather than failing on first use:

```go
// internal/core/config/config.go
type Config struct {
    DatabaseURL         string `env:"DATABASE_URL,required"`
    AnthropicAPIKey     string `env:"ANTHROPIC_API_KEY,required"`
    LLMDailyTokenBudget int    `env:"LLM_DAILY_TOKEN_BUDGET" envDefault:"2000000"`
}

// Load reads and validates configuration. It returns an error rather than
// exiting so the caller controls startup ordering.
func Load() (*Config, error) {
    var c Config
    if err := env.Parse(&c); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    if c.LLMDailyTokenBudget <= 0 {
        return nil, errors.New("LLM_DAILY_TOKEN_BUDGET must be positive")
    }
    return &c, nil
}
```

Never call `os.Getenv` outside this package. Never store a secret in a struct
that has a `String()` method or gets logged whole — Go will happily print your
API key inside a `%+v`.

## Layout

```
cmd/<service>/main.go        wiring only
internal/
  transport/http/            handlers — decode, delegate, encode
  service/                   business logic, owns transactions
  repository/                data access, returns domain types
  domain/                    types and errors, no dependencies
  agent/                     agent nodes and tool calls
  core/config, core/log      configuration and logging
```

`internal/` so nothing here becomes an accidental public API. `domain` imports
nothing from the other packages — if it needs to, the dependency is pointing the
wrong way.

## Errors

```go
// internal/domain/errors.go
type AppError struct {
    Code    string
    Message string
    Status  int
    Detail  map[string]any
    err     error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.err }

var (
    ErrNotFound         = &AppError{Code: "not_found", Status: 404}
    ErrApprovalRequired = &AppError{Code: "approval_required", Status: 409}
)

// NotFound returns a not-found error for the named resource.
func NotFound(resource string, err error) *AppError {
    return &AppError{
        Code: "not_found", Status: 404,
        Message: resource + " not found",
        Detail:  map[string]any{"resource": resource},
        err:     err,
    }
}
```

Wrap with `%w` at every layer boundary so the chain survives to the handler:

```go
if err := r.db.QueryRow(ctx, q, id).Scan(&lead); err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domain.NotFound("lead", err)
    }
    return nil, fmt.Errorf("load lead %d: %w", id, err)
}
```

One place maps `*AppError` to a response, so the error contract lives in a
single file:

```go
func writeError(w http.ResponseWriter, r *http.Request, err error) {
    var appErr *domain.AppError
    if errors.As(err, &appErr) {
        log.WarnContext(r.Context(), "app_error", "code", appErr.Code)
        writeJSON(w, appErr.Status, errorBody{Code: appErr.Code, Message: appErr.Message})
        return
    }
    log.ErrorContext(r.Context(), "unhandled", "err", err)
    writeJSON(w, 500, errorBody{Code: "internal_error"})
}
```

Never `panic` in library or service code. A panic in one request handler takes
the process with it unless something recovers, and a recover that swallows the
panic hides a bug you needed to see. Panic only in `main` during startup, where
failing loudly is correct.

## Context

Every function that does I/O takes `ctx context.Context` as its first parameter
and passes it down. This is what makes a cancelled request stop doing work
instead of finishing an expensive LLM call nobody is waiting for. `contextcheck`
in the linter config catches the ones that get missed.

Never store a `context.Context` in a struct field.

## Handler

```go
// internal/transport/http/leads.go

// Implements: #142
func (h *Handler) ListLeads(w http.ResponseWriter, r *http.Request) {
    user, ok := auth.UserFrom(r.Context())
    if !ok {
        writeError(w, r, domain.Unauthorized())
        return
    }
    q, err := decodeListQuery(r)
    if err != nil {
        writeError(w, r, err)
        return
    }
    page, err := h.leads.ListForUser(r.Context(), user, q.Cursor, q.Limit)
    if err != nil {
        writeError(w, r, err)
        return
    }
    writeJSON(w, http.StatusOK, toLeadPage(page))
}
```

Decode, delegate, encode. No SQL, no business rule.

## Logging

`log/slog`, never `fmt.Print*`. Context-aware calls so correlation fields ride
along:

```go
log := slog.With("run_id", runID, "agent_id", nodeName)
log.InfoContext(ctx, "tool_call", "tool", "crm.read_contact")
```

## Data layer

`pgx` with `sqlc`-generated queries, or hand-written `pgx` calls. Not an ORM.

- Migrations in `migrations/`, applied by `golang-migrate`, with a real `.down.sql`.
- Never build SQL with `fmt.Sprintf`. `$1` placeholders only. `gosec` flags the
  obvious cases and misses the clever ones, so this is also a review item.
- `defer rows.Close()` immediately after the query, and check `rows.Err()` after
  the loop. A missed `rows.Err()` silently truncates a result set on a mid-scan
  error, which looks like "the data just isn't there".
- Transactions owned at the service layer:

```go
func (s *LeadService) Import(ctx context.Context, batch []domain.Lead) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin: %w", err)
    }
    defer tx.Rollback(ctx) // no-op after a successful commit
    if err := s.repo.WithTx(tx).InsertMany(ctx, batch); err != nil {
        return fmt.Errorf("insert batch: %w", err)
    }
    return tx.Commit(ctx)
}
```

## Agent nodes in Go

Go is not the orchestrator — LangGraph runs in Python or TypeScript. Go here is
for tool servers and workers a node calls. The contracts still apply to the
tool side:

```go
// internal/agent/tools/crm.go

// AllowedCallers lists the node names permitted to invoke this tool. Enforcing
// the allowlist on the tool side as well as the node side means a
// misconfigured node cannot reach a tool it was never granted.
var AllowedCallers = []string{"draft_followup", "research_account"}

const TokenBudget = 0 // no LLM call in this tool
```

Every tool call is logged with `run_id`, `agent_id`, and the tool name before it
executes, so the audit log records the attempt and not merely the successes.

Any tool that performs an irreversible action verifies a named approval and
refuses without one:

```go
// Implements: #187
func (t *EmailTool) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
    approval, err := t.approvals.Get(ctx, req.RunID, "send_followup")
    if err != nil {
        return nil, fmt.Errorf("load approval: %w", err)
    }
    if approval == nil || approval.Approver == "" {
        return nil, domain.ApprovalRequired("send_followup", req.RunID)
    }
    ...
}
```

## Testing

Table-driven, with the failure cases first:

```go
func TestListForUser(t *testing.T) {
    tests := []struct {
        name    string
        limit   int
        wantErr error
    }{
        {"rejects zero limit", 0, domain.ErrValidation},
        {"rejects negative limit", -1, domain.ErrValidation},
        {"accepts valid limit", 50, nil},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            _, err := svc.ListForUser(ctx, user, "", tc.limit)
            if !errors.Is(err, tc.wantErr) {
                t.Fatalf("got %v, want %v", err, tc.wantErr)
            }
        })
    }
}
```

Integration tests against a real Postgres (testcontainers or a CI service
container), each in a transaction that rolls back. Never mock the database.

Coverage floors come from `COVERAGE_FLOOR` and `AGENT_COVERAGE_FLOOR` in
`scripts/gate.py` — currently 100% overall and 100% on `internal/agent/`.
They are one number for all three stacks on purpose; a per-language floor is
a per-language argument every quarter.

Lowering a floor for one path is a dated decision, not a flag: add a block to
`.coverage-exemptions.toml` with `path`, `floor`, `reason`, `owner` and
`expires`, the same shape as a security suppression. The gate fails on an
expired exemption, and names any exemption that covers agent code so review
sees it. Without an expiry, "we will cover it later" becomes permanent.

## Never

- `fmt.Print*` in application code.
- A discarded error (`_ = doThing()`) without a comment explaining why it cannot
  matter.
- `panic` outside startup.
- `os.Getenv` outside `core/config`.
- SQL built with `fmt.Sprintf`.
- `context.Context` in a struct field.
- A goroutine with no way to stop it. If it does not select on `ctx.Done()` or
  a close channel, it is a leak.
