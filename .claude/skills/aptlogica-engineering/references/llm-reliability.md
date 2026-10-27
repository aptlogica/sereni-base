# LLM reliability — grounding, context, and cost

Read this when building or reviewing any code path that calls a model. Rules
live in SKILL.md §5; this file is how to satisfy them and why each one exists.

## What does not work

Before the controls, the anti-patterns — because they are what teams reach for
first and they consume effort without reducing risk.

**Instructions to be accurate.** "Be factual", "do not hallucinate", "only use
information from the context" are worth including but they are not controls.
They shift a distribution slightly. They do not prevent anything, and treating
them as mitigation means shipping with no mitigation.

**Self-reported confidence.** Asking a model how confident it is returns a
plausible number weakly correlated with correctness. The danger is not that it
is useless — it is that it looks useful. Put a `confidence` field in a node
output and within a month something will threshold on it, and we will have a
governance mechanism that is decorative. Confidence must come from retrieval
agreement, a separate verifier, or a deterministic check. Never from asking.

This is a review item rather than a gate check, because the pattern is too
varied to detect reliably. **In review: reject any `confidence`, `certainty`, or
`likelihood` field populated from model output.**

**Temperature as a correctness knob.** Temperature 0 makes output more
repeatable, not more true. A confidently wrong answer is stable at temperature 0.

## Grounding

The failure mode that matters for any LLM-backed feature is not gibberish. It is a fluent,
well-formatted email to a client stating a commitment nobody made. Everything
here targets that.

### Provenance on claims

Any node whose output will be read as fact — a draft message, a summary, a
research brief, a lead record — returns structured claims, not prose:

```python
class Claim(BaseModel):
    """One factual assertion with the retrieved span it came from."""

    text: str
    source_id: str  # id of a retrieved document or record
    span: tuple[int, int] | None = None  # char offsets within that source


class DraftOutput(BaseModel):
    body: str
    claims: list[Claim]
```

A validator then refuses output containing an assertion with no `source_id`.

**Decision made, and reversible:** the standard requires claim-bearing nodes to
carry provenance in their return type, and the gate checks that such modules
reference a claims structure. It does not attempt to verify that every sentence
in `body` maps to a claim — that needs a verifier pass (below), not a regex.
The stricter alternative — every node returning `Claim[]` and prose assembled
only from claims — is more invasive and much harder to bypass. Worth revisiting
once a few nodes exist, and far cheaper to adopt before eighteen do.

### No invented identifiers

Every email address, URL, phone number, price, date, and record ID appearing in
generated output must appear in the retrieved set. This is set membership, so it
is cheap and deterministic:

```python
def check_identifiers(output: str, sources: list[str]) -> list[str]:
    """Return identifiers in output that appear in no source."""
    corpus = "\n".join(sources)
    found = IDENTIFIER_RE.findall(output)
    return [ident for ident in found if ident not in corpus]
```

Run it before anything reaches a human for approval. It catches the most
embarrassing class of error — a plausible-looking email address for a real
person — and it never produces a false negative on the cases that matter.

### A verifier separate from the generator

A second, cheap pass that reads the draft and its sources and reports
unsupported claims. Same principle as human review: do not let the generator
mark its own homework.

```python
VERIFIER_MODEL = "claude-haiku-4-5"  # cheap; this is a checking task


async def verify(draft: DraftOutput, sources: list[Source]) -> list[str]:
    """Return claims the sources do not support."""
```

This costs an extra call per draft, which is a real trade. Apply it where the
output leaves the building — outbound email, client documents, anything
published. Skip it on internal summaries a human is about to read anyway.

### Abstention

The model must be able to decline. If "insufficient context" is not an available
outcome, it fills the gap — that is not a flaw in the model, it is what we asked
for by leaving no other option.

No new machinery needed: this is `blocked` with a reason, which SKILL.md §3's
seven-state vocabulary already covers.

```python
if not sources:
    await emit_event(run_id, NODE_NAME, "blocked", reason="no_sources_retrieved")
    return {"status": "insufficient_context"}
```

Abstention is a success. If it is counted as failure in any metric or eval, that
metric is training the system to guess.

### Staleness

Retrieved data carries an as-of timestamp, and output that depends on it says
so. "Their Series B closed last month" is wrong six months later, and nothing in
the pipeline currently knows that. Any claim derived from a source older than
its configured freshness window is downgraded or refused.

## Structured output

Free text where a field is expected is where fabrication enters, because there
is nothing to validate against.

- Every generation path parses into a Pydantic model (Python), a Zod schema
  (Node), or a struct with explicit validation (Go).
- Retry on parse failure, capped — two attempts, then fail loudly. An uncapped
  retry loop on a malformed response is both a cost incident and an outage.
- A parse failure is an `error` event, not a silent fallback to the raw string.

```python
for attempt in range(MAX_PARSE_RETRIES):
    raw = await llm.complete(prompt, budget=TOKEN_BUDGET, run_id=run_id)
    try:
        return DraftOutput.model_validate_json(raw)
    except ValidationError as exc:
        last = exc
await emit_event(run_id, NODE_NAME, "error", reason="unparseable_output")
raise ModelOutputError("draft did not validate after retries") from last
```

## Context management

Context length is where quality degrades, and it degrades quietly. This matters
more than token cost.

### Context ceiling per node

Declared alongside the other node constants, and distinct from `TOKEN_BUDGET`:

```python
ALLOWED_TOOLS = ("crm.read_contact",)
TOKEN_BUDGET = 12_000  # spend ceiling — about money
CONTEXT_CEILING = 40_000  # input ceiling — about attention
MODEL = "claude-haiku-4-5"
```

`TOKEN_BUDGET` stops a node costing more than it should. `CONTEXT_CEILING` stops
it being fed more than it can attend to. They are different failures and need
different limits.

### Truncation is explicit or it is a bug

Silent truncation is a correctness failure: the model answers confidently on
partial data with no way to know data was removed. Either mark it visibly or
refuse.

```python
if tokens(context) > CONTEXT_CEILING:
    context = truncate(context, CONTEXT_CEILING)
    context += "\n[TRUNCATED: some retrieved content was omitted]"
    await emit_event(run_id, NODE_NAME, "planning", truncated=True)
```

If the truncated part was load-bearing, prefer abstention over a marked guess.

### Bounded state

LangGraph state grows across checkpoints, and unbounded state is the main cause
of degradation on long runs. Every graph declares explicitly what is carried
forward, what is summarized, and what is dropped. Undeclared means dropped.

**A summary cannot be cited.** Once a turn is summarized it is lossy, so it must
not serve as a `source_id`. A claim traced to a summary is a claim traced to
nothing.

**The transcript is not memory.** Durable facts go to the repository's own
store, with a schema and a migration — Postgres in the Python services, SQLite
in a single-operator runner. Which engine is a project decision; that a fact
survives outside the conversation is not.
Using accumulated conversation as the store of record is the mistake that makes
long-running agents unpredictable, and it is exactly what the large community
skill catalogs do at scale.

## Token efficiency

Cost control is a design property, not a dashboard.

**Per-node model selection.** Declared as `MODEL` above. Most agents do not
need the expensive model — classification, extraction, routing and verification
are small-model work; drafting and multi-step reasoning are not. Routing
by node is the single largest saving available, usually more than every prompt
optimization combined.

**Prompt caching on the stable prefix.** System prompt plus this standard is
large, stable, and repeated on every call. Cache it. Order the prompt so the
stable part comes first and the volatile part last, or the cache never hits.

**Retrieve spans, not documents.** Stuffing a whole document costs tokens and
buries the relevant part in the middle of the context, which is where attention
is weakest. Retrieve and pass the span.

**Measure cost per run, not per call.** A per-call budget with no run-level
rollup lets a retry loop spend unboundedly while every individual call stays
inside its limit. The run-level number is the one that appears on the invoice.

**Do not pay twice for the same tokens.** If a node has already retrieved a
document, pass the reference through state rather than re-retrieving it in the
next node.

## Development-time reliability

This section is about how an agent — including Claude Code — should behave while
writing this code. The failure modes are different from runtime and just as
expensive.

**Verify APIs against what is installed, not from memory.** Model knowledge of
library signatures is confidently wrong often enough to matter, especially for
fast-moving libraries like LangGraph and SQLAlchemy 2.x. Read the actual
signature out of the environment:

```bash
python -c "import langgraph.graph as g; help(g.StateGraph.add_node)"
pnpm why @langchain/langgraph
go doc github.com/jackc/pgx/v5.Conn.Query
```

**Never invent a dependency.** Check the lockfile before importing. A plausible
package name that does not exist is a five-minute detour; a plausible package
name that exists and is malicious is an incident.

**Read before asserting.** Do not claim a file, test, function or config value
exists without opening it. Cite `file:line`. "The existing handler already does
this" is a claim requiring evidence like any other.

**Never report that the gate passed without running it.** This is the
load-bearing one, and the reason the gate exists at all. Paste the exit code.

**Flag citation uncertainty.** Any reference to a paper, RFC, doc page or blog
post may be fabricated or misremembered. Say so, and prefer a command that
checks over a citation that asserts.
