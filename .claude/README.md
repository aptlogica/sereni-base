# Claude Code setup for SereniBase

Five skills under `.claude/skills/` and three standard-library Python scripts
under `scripts/`. The scripts work without Claude Code; the skills tell Claude
when and how to run them.

| Skill | Use for | Script |
|---|---|---|
| `aptlogica-engineering` | Implementing, refactoring, upgrading, debugging | `scripts/gate.py` |
| `aptlogica-review` | Read-only review of a diff, branch or PR, with a verdict | reads `reports/gate.json` |
| `aptlogica-documentation` | GoDoc comments, file headers, `Called from:` lines | `scripts/docgate.py` |
| `aptlogica-memory` | "Where is X / what calls Y", with `file:line` answers | `scripts/memory.py` |
| `aptlogica-code-graphs` | Architecture maps (graphify fork, Apache-2.0; keep `LICENSE` and `NOTICE`) | — |

## Quick start

```bash
python scripts/memory.py                            # build the local index (.claude/memory/)
python scripts/gate.py --baseline                   # record existing failures
python scripts/gate.py --report reports/gate.json   # format, vet, lint, race tests, vulncheck, secrets
python scripts/docgate.py --stack go                # undocumented exports
```

On Windows set `PYTHONUTF8=1`, or the summary line can fail with
`UnicodeEncodeError`. The gate needs `golangci-lint`, `govulncheck` and
`gitleaks` on `PATH`, and exits `2` without them.

## SereniBase-specific configuration

The scripts are shared with other Aptlogica repositories. Lines changed for this
repository are marked `# SereniBase`:

- `internal/handlers`, `internal/router`, `internal/middleware` are the transport layer;
  `internal/services` the service layer; `internal/models` models; `internal/dto` schemas;
  `internal/providers` core adapters.
- `memory.py` indexes `internal/` and `cmd/` by default (`--paths tests` for the
  tests). Go methods are indexed as `Type.Method`, grouped `const`/`var`/`type`
  members are included, and the license notice is never used as a file header.
- Traceability accepts `Implements: #<issue>` as well as the internal EPIC/FR form.
- The gate's Go test command matches CI: `-coverpkg=./internal/... ./tests/...`.
  It writes `reports/coverage.out` (the root `coverage.out` is tracked for Sonar)
  and enforces `COVERAGE_FLOOR` on it.

Each script exists twice: in `.claude/skills/<skill>/scripts/` (canonical) and
in `scripts/`. Keep them identical.

## What is not committed

`.claude/memory/`, `reports/`, `graphify-out/` and `.claude/settings.local.json`
are gitignored. Each developer generates them locally.

## Security

`.claude/` and `scripts/` are instructions and code that Claude runs. Changes to
them need a code-owner review (see `.github/CODEOWNERS`). Don't run Claude Code
on an untrusted PR branch that touches these paths; review the diff from
`master` instead.
