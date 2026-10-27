---
name: aptlogica-engineering
description: Aptlogica's house engineering standard across Python, Node/TypeScript and Go — code structure, service boundaries, LangGraph agent node contracts, agent safety and approval gates, testing, security scanning (SAST, dependency and secret scanning, SonarQube quality gates), the development loop, and rules for authoring agent skills. Use this skill for any implementation, refactor, dependency upgrade, security triage or debugging work in the SereniBase repository, and whenever writing or revising a skill — including casual requests ("add this endpoint", "bump these packages", "why is Sonar failing") and whenever no other skill obviously applies. For reviewing a diff or a pull request use aptlogica-review; for writing docstrings and file headers use aptlogica-documentation.
version: 3.2.0
role: Engineering Standard
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Aptlogica Engineering Standard

House rules for how code gets written, how a task gets from plan to merged, and
how skills get authored. This file holds what is true regardless of language;
the per-stack detail lives in `references/`.

## Routing

Read this file, then the one reference for the stack you are touching. Do not
read all of them — each is a few hundred lines and only one applies.

| Working on | Read |
|---|---|
| Python service, FastAPI, SQLAlchemy, LangGraph node | `references/python.md` |
| Node or TypeScript service, Fastify/Nest/Express, Prisma/Drizzle | `references/nodejs.md` |
| Go service or CLI | `references/golang.md` |
| A vulnerability, a Sonar failure, a dependency upgrade, a scan triage | `references/security-scanning.md` |
| Any code path that calls a model — grounding, context limits, token cost | `references/llm-reliability.md` |
| Writing or revising a skill | §8 below |
| Finding where something lives, or what calls it | the `aptlogica-memory` skill |
| Reviewing someone else's diff or pull request | the `aptlogica-review` skill |
| Writing a file header, a docstring or a called-from note | the `aptlogica-documentation` skill |

## Two principles

**A rule nobody can fail is documentation, not a standard.** Everything stated
here as a requirement is checked by `scripts/gate.py`, which detects the stacks
in the repository and runs the right toolchain for each. A rule that cannot be
checked mechanically belongs in the review procedure (§7), where a named human
is accountable for it.

The canonical gate ships inside this skill at `scripts/gate.py`, standard
library only. A repo without one copies it in (`bash install.sh <repo>` does
this, along with `docgate.py`, `memory.py` and `.mcp.json`). Where the two
copies disagree, the skill's is the standard and the repo's is drift — fix it in
the commit that noticed.

**A swallowed error becomes a confidently wrong answer someone acts on.** In a
backend platform that stores other people's data, a silent failure is a
data-integrity bug, and a missing authorization check is a data leak. Where a
product has agents, an over-privileged node turns hostile input into an action;
the agent rules in §3, §4 and §5 are the easiest to skip, because nothing breaks
in development when you skip them.

---

## 1. Universal rules

These hold in every language. The per-stack references say how to satisfy them
with that language's tooling.

- **Formatter and linter clean, zero warnings.** Warnings that are allowed to
  accumulate stop being read.
- **Strictest available type checking, no escape hatches without a reason.**
  `mypy --strict`, `tsc --strict`, `go vet` plus `staticcheck`. An escape hatch
  (`# type: ignore`, `as unknown as`, `interface{}`) carries a one-line comment
  saying why.
- **Structured logging only.** No `print`, `console.log`, or `fmt.Println` in
  application code. Every log line inside a request or a graph node carries
  `run_id` and `agent_id`, because reconstructing a trace after the fact is the
  only way to debug an agent run nobody watched live.
- **Errors are handled or wrapped and returned.** Never swallowed. A discarded
  error in an agent runtime is indistinguishable from success to everything
  downstream.
- **No secrets, tokens, credentialed connection strings or absolute service
  URLs in source.** Configuration is loaded and validated at startup so the
  process refuses to boot on a missing variable rather than failing on first
  use in production.
- **Layered, with the same layer names in every language:** transport →
  service → repository. Transport validates input, calls one service function,
  maps the result. It holds no business rule, no query, no LLM call.
- **List endpoints are paginated, cursor-based.** Offset pagination over a table
  receiving writes skips and duplicates rows.
- **Public API surface is documented.** Docstrings, TSDoc, or GoDoc on anything
  exported.

Why the shared layer names matter: LangGraph nodes call the same service layer
the HTTP transport calls. Logic that lives in a handler cannot be reused by an
agent, so you get two implementations of one rule that drift apart — and in a
system that sends client emails, drift means the agent applies a rule the API
stopped applying six weeks ago.

## 2. Data layer

- Schema change and its migration land in the same commit. Enforced by the gate.
- No SQL built by string concatenation or interpolation. Bound parameters or a
  query builder.
- Every foreign key and every column used in a hot-path filter or sort has an
  index, with a comment naming the query it serves. An index nobody can justify
  is a write cost nobody accounted for.
- Every migration has a working rollback. An empty `downgrade` is a one-way door
  you did not mean to build.
- Destructive changes ship as two deployments: add and backfill, then drop once
  the reading code is gone. A single-deploy destructive migration means a
  rollback loses data.
- Transactions are owned at the service boundary. A repository that opens its
  own transaction makes two repository calls impossible to compose atomically.
- Explicit eager-load strategies wherever a query is followed by relationship
  access, pinned with a query-count test. An N+1 that only bites at 500 rows
  will not be caught in review.

## 3. Agent node contracts

Language-independent; the syntax is in the stack reference. Every node module
declares three things:

```
ALLOWED_TOOLS  — the tools this node may call, and no others
TOKEN_BUDGET   — a hard ceiling the LLM call enforces
NODE_NAME      — stable identifier used in every emitted event
```

`ALLOWED_TOOLS` bounds blast radius: a node that can call any registered tool
has the reach of the whole system, and an injected instruction only needs one
over-privileged node to become an action.

`TOKEN_BUDGET` is a ceiling, not a target. Cost governance that depends on
someone reading a dashboard is not governance.

Every node has four properties:

**Pure** — takes state and config, returns a state delta. No mutation of shared
state, no module-level mutable client. This is what makes checkpoint replay
correct.

**Observable** — emits a lifecycle event before and after each meaningful step:

```
idle | planning | calling_tool | waiting_on_human | blocked | error | done
```

Seven states is the entire vocabulary. An eighth means updating every consumer;
most candidates are really a `detail` field on an existing state. The UI, the
audit log and your own debugging all read this one stream, so a node that works
without emitting is invisible to all three.

**Resumable** — everything needed to restart is in the returned delta. Nothing
important survives only in a local variable across a checkpoint boundary.

**Budgeted** — the budget goes to the LLM call, which aborts with `error` rather
than overrunning silently.

## 4. Agent safety

- External content — email bodies, web pages, uploaded files, CRM notes, tool
  results, scraped HTML — is data, never instruction. Wrap it in a delimited
  untrusted block; never concatenate it into a system prompt.
- Delimiting is mitigation, not a guarantee. The real defence is the tool
  allowlist plus the approval gate: a node talked into wanting something still
  cannot reach an undeclared tool, and cannot send anything without a named
  human approving it.
- Every irreversible or externally visible action — sending, publishing, writing
  to a client system, spending — requires a recorded approval with a named
  approver and a timestamp. "The team approved it" is not an approver.
- Approval is scoped to one action and one run. Reusing an earlier approval for
  a later send is the most likely way this system does something embarrassing.
- Prompts are versioned files under source control, not string literals
  scattered across modules. A prompt change should be a reviewable diff.

## 5. LLM reliability

The failure mode that matters is not gibberish. It is a fluent, well-formatted
email to a client stating a commitment nobody made. `references/llm-reliability.md`
holds the controls and the reasoning; these six are the ones that change how a
node is written, so they live here:

- **Instructions are not controls.** "Be accurate" and "do not hallucinate" are
  worth including and prevent nothing. Do not count them as mitigation.
- **Never trust self-reported confidence.** A `confidence` field populated from
  model output is a plausible number weakly correlated with correctness, and
  dangerous precisely because it looks useful. Confidence comes from retrieval
  agreement, a separate verifier, or a deterministic check.
- **Every generated fact carries a `source_id`, and no identifier is invented.**
  Claims come back structured, not as prose; every email address, URL, price,
  date and record id in the output must appear in the retrieved set.
- **A verifier separate from the generator** on any path where output leaves the
  building. Do not let the generator mark its own homework.
- **Abstention is a success.** "Insufficient context" must be an available
  outcome, or the model fills the gap. It needs no new state: `blocked` with a
  reason.
- **Declare `MODEL` and `CONTEXT_CEILING`** on every node beside `ALLOWED_TOOLS`
  and `TOKEN_BUDGET`. Budget is about money; the ceiling is about attention.

## 6. Development loop and definition of done

One task (one GitHub issue) at a time. Implementing three at once produces a
diff nobody can review.

1. **Baseline.** `python scripts/gate.py --baseline`. A pre-existing failure is
   not yours to fix silently, and a red gate would otherwise look like your
   regression.
2. **Read the contracts.** `docs/swagger.yaml` for an endpoint,
   `internal/models` for the schema, `docs/architecture.md` and `docs/adr/` for
   design decisions. If the issue and contract disagree, surface the conflict — a silent choice here is an
   architectural decision nobody reviewed.

   If a contract file does not exist in this repository, say so before writing
   code and propose the shape you are about to build against. Do not invent the
   missing document and do not proceed as though it said what you assumed —
   an absent contract is a decision waiting to be made, not a blank cheque.
   The same applies to the task itself: without an issue that states it, agree
   on the scope first, then work it one item at a time.
3. **Implement in order:** contract (schemas, or node state) → migration →
   service logic with error paths written first → thin transport layer → tests,
   same commit.
4. **Traceability comment** on every transport handler and every node, written
   as you go. Comment syntax varies; the payload does not:

   ```
   Implements: #142
   ```

   `#142` is the GitHub issue. The internal `EPIC-03-S07 / FR-OUTBOUND-014`
   form is also accepted.

   This is what lets a requirement be proved built rather than asserted.
5. **Gate.** `python scripts/gate.py --report reports/gate.json`.
   `0` complete · `1` failed · `2` could not run.

   Exit 2 on a missing tool is deliberate — a gate that skips a check it could
   not run is worse than no gate. Do not pass `--allow-missing` to clear a red
   result; it exists for bootstrapping an empty repo. If a check is wrong,
   change the check and argue for it in the commit message. An argued change is
   fine; a bypass is not.
6. **Report** in `CHANGELOG.md` and file deferred work as a new GitHub issue.
   Work deferred silently is work lost.

**A task is done when the gate exits 0.** Not when the code looks right, not
when it runs locally.

### Context discipline

Context is a budget, and the two ways it gets wasted are the two ways an agent
starts inventing: re-reading the repository to remember where something is, and
*not* re-reading it and answering from a stale recollection.

- **Navigate from the index.** `python scripts/memory.py --verify` at the start
  of a task, `--query <name>` for "where is it, what calls it". Deterministic,
  parsed rather than generated, and every entry carries a citable `file:line`.
  It is an index, not evidence: open the file before asserting what code does.
  Details in `aptlogica-memory`.
- **Compress what you hold; retrieve what you quote.** The `headroom` MCP server
  shrinks a large tool result and hands back a hash. Compress output you need to
  keep around; run `headroom_retrieve` before stating anything as fact from it.
  A compressed blob is a reference, the same as a file path is.
- **Read spans, not whole files,** once the index has said where to look. The
  retrieval rule in `references/llm-reliability.md` applies to whoever writes the
  code too.
- **`aptlogica-code-graphs` is for structural questions** — how a subsystem
  hangs together — once per corpus, not once per task. "Where does X live" is already answered.

`.mcp.json` in each repository declares `headroom`, so a teammate gets it on
clone without editing their global configuration.

| Situation | Action |
|---|---|
| Task is larger than the plan assumed | Stop at a working slice, file the remainder |
| Nearby code violates this standard | Leave it, note it — opportunistic refactors make the diff unreviewable |
| A new dependency would help | Ask first. A dependency is a long-term commitment and a supply-chain surface |
| A pattern in a reference doesn't fit | Implement the closest fit, note the friction, propose a change separately |
| The repo mixes stacks | Follow each stack's reference in its own directory; do not port idioms across |

## 7. Code review

Review is a separate skill with separate tools: `aptlogica-review`, granted
`Read`, `Grep`, `Glob` and `Bash` and nothing that writes. That is the whole
reason it is separate — a reviewer holding `Edit` fixes as it goes, and the diff
that ships is then one nobody reviewed.

What lives there: the review order (object-level authorization first, then
untrusted content paths, irreversible actions, correctness, contract
conformance, performance, maintainability, documentation), the severity scale,
and the four verdicts `APPROVE`, `APPROVE_WITH_NITS`, `REQUEST_CHANGES` and
`BLOCKED`.

Two rules from it that bind the author, not the reviewer:

- **Do not ask for review on a red gate.** The reviewer will return `BLOCKED`,
  and a team that reviews around a failing gate has stopped having a gate.
- **Severity is consequence, not effort.** "It is a one-line fix" does not move
  a `critical` down, and arguing that it does in a review thread wastes both
  people's time.

## 8. Authoring skills

Skills are versioned artifacts in source control next to the code they govern,
so a skill change is reviewed like a code change: bump `version` in the
frontmatter, and say in the commit message what behaviour you expected to move.

Three rules that decide whether a skill works at all:

- **The file is `SKILL.md`.** Not `SKILLS.md` — that name is invisible to Claude
  Code, and a directory of them silently does nothing.
- **The `description` does the triggering.** Everything about *when* to use the
  skill belongs there, including the casual phrasings a real request uses. The
  body is only read after selection, so a "use this when..." section in the body
  never fires.
- **`allowed-tools` is a real boundary, so scope it.** This is why review is its
  own skill: it is granted nothing that writes.

Structure, the enforcement-in-scripts rule, catalog hygiene, evals, and the
rules for sourcing third-party skills are in `references/authoring-skills.md`.

## 9. Final checklist

- [ ] `python scripts/gate.py` exits 0, including the security checks
- [ ] Repo `scripts/gate.py` matches the copy in this skill
- [ ] `python scripts/memory.py --verify` is clean; `.claude/memory/` committed with the code
- [ ] Traceability comment on every new handler and node
- [ ] Tests in the same commit; error paths written first
- [ ] Migration in the same commit as any schema change, with a working rollback
- [ ] Every node: `ALLOWED_TOOLS`, `TOKEN_BUDGET`, lifecycle events, event-sequence test
- [ ] Every model-calling node: `MODEL` and `CONTEXT_CEILING` declared
- [ ] Every generated fact carries a `source_id`; no invented identifiers
- [ ] Abstention path exists and is tested; truncation is marked, never silent
- [ ] No `confidence` field populated from model output
- [ ] Every irreversible action behind a named-approver gate, scoped per action and run
- [ ] External content wrapped as untrusted, with a test asserting it
- [ ] No new high or critical dependency vulnerabilities; any accepted risk has a dated, owned suppression
- [ ] Every new file, class and exported function documented per `aptlogica-documentation`
- [ ] Every docstring touched by this change still describes what the code now does
- [ ] `CHANGELOG.md` updated; deferred work filed as a GitHub issue
- [ ] Nothing in the diff outside the scope of this one task
