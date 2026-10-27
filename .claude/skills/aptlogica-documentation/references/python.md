# Python — docstrings

Read this with SKILL.md's four things. This file is only the syntax and the
Python-specific traps.

## Style

PEP 257 shape, no reST/Sphinx field lists. The type hints carry the types; a
`:param x: the x` line is a second copy that rots separately from the first.

```python
def f(...) -> T:
    """One line, imperative mood, ends with a period."""
```

A one-line docstring stays on one line. A longer one gets a summary line, a
blank line, then the detail — never detail on the summary line.

`mypy --strict` is already enforcing the signature, so the docstring's job is
everything the signature cannot say: ordering, invariants, side effects,
raises, callers.

## Module header

Every module under `src/` opens with one, naming the layer:

```python
"""Lead persistence. Repository layer — owns no transaction, returns domain
objects, never ORM rows.

Called from: services.leads, services.enrichment.
"""
```

`__init__.py` gets a package header instead: what the package is for, and which
of its modules a newcomer should open first.

```python
"""Outbound agent nodes.

Start at `graph.py` for the wiring; each node module is self-contained and
declares its own ALLOWED_TOOLS.
"""
```

## Classes

```python
class LeadRepository:
    """Reads and writes leads for one org.

    Holds an `AsyncSession`, so one instance per request or per graph node.
    Never a module-level singleton — a shared session leaks a transaction
    across requests.
    """
```

Pydantic schemas and dataclasses usually need one line or none: the field names
and types are the documentation. Document the *unobvious field*, on the field:

```python
class LeadCreate(BaseModel):
    email: EmailStr
    # Free text from the import CSV — untrusted, never interpolated into a
    # prompt. See SKILL.md §4 in aptlogica-engineering.
    source_note: str | None = None
```

## Functions

```python
async def list_for_org(
    self, org_id: UUID, cursor: str | None, limit: int
) -> Page[Lead]:
    """Return one page of an org's leads, newest first.

    Cursor-based: offset pagination over this table skips rows while writes
    land. Raises `InvalidCursor` when the cursor does not decode.

    Called from: api.leads.list_leads, agents.outbound.select_targets.
    """
```

Document the raise when it is part of the contract. Do not list every exception
that could theoretically escape.

Private helpers (`_name`) under about five lines get a good name, not a
docstring. A private helper that needs a paragraph is a service function that
has not been extracted yet.

## Generators, context managers, async

Say what is yielded and when cleanup happens — the signature does not.

```python
@asynccontextmanager
async def unit_of_work() -> AsyncIterator[AsyncSession]:
    """Yield a session inside one transaction, committing on clean exit.

    Rolls back on any exception, including `CancelledError`, so a cancelled
    agent run leaves no half-written state.

    Called from: services.leads, services.outbound, and 6 others.
    """
```

## LangGraph nodes

Node modules carry the three extra lines from SKILL.md, mirroring the
declarations rather than restating them from memory:

```python
"""Draft an outbound reply from retrieved context. Agent node.

Tools:   read_crm, search_docs            (mirrors ALLOWED_TOOLS)
Budget:  40_000 tokens, ceiling 120_000   (mirrors TOKEN_BUDGET, CONTEXT_CEILING)
Emits:   planning → calling_tool → done, or blocked when context is thin

Called from: agents.outbound.graph.
"""

ALLOWED_TOOLS = ("read_crm", "search_docs")
TOKEN_BUDGET = 40_000
CONTEXT_CEILING = 120_000
NODE_NAME = "outbound.draft"
```

The node function itself documents the state delta — which keys it reads and
which it returns — because that is the contract the graph depends on and the
type alias rarely spells it out:

```python
async def draft(state: OutboundState, config: RunnableConfig) -> OutboundState:
    """Draft a reply from `state.retrieved` and return `draft` + `claims`.

    Reads `retrieved`, `thread_id`. Returns only the keys it changed. Returns
    `blocked` with a reason rather than drafting when `retrieved` is empty —
    abstention is a success, not a failure path.

    Implements: EPIC-05-S02 / FR-COMMS-031
    """
```

The `Implements:` line is the traceability comment from the engineering
standard. It lives inside the docstring for nodes and handlers; it is not a
substitute for the summary line.

## Tests

Test functions are documented by their names. `test_get_lead_cross_org_returns_404`
needs no docstring; `test_lead_2` needs a rename, not a docstring.

The exception is a regression test, which records the thing that broke:

```python
def test_cursor_survives_concurrent_insert() -> None:
    """Regression: offset pagination duplicated row 51 under concurrent writes."""
```
