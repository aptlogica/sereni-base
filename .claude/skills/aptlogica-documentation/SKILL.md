---
name: aptlogica-documentation
description: Aptlogica's in-code documentation standard for Python, Node/TypeScript and Go — what a file header says, the one-line summary every class and exported function carries, how to record where something is called from, what to leave undocumented, and how to keep a docstring from going stale. Use this skill whenever writing or fixing comments, docstrings, TSDoc or GoDoc, when asked "what does this file/function/class do", when asked to document, annotate or add headers to code, and when a review finds a missing or wrong docstring. The generator is scripts/docgate.py, which lists every undocumented export and exits non-zero.
version: 1.2.0
role: Documentation Standard
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Aptlogica Documentation Standard

Documentation in this codebase exists for one reader: someone — a person or an
agent — who has landed in a file they did not write and needs to know, in a few
seconds, what it holds, what calls it, and what will break if they change it.

That reader does not want prose. Every rule below is about producing the
smallest thing that answers the question.

## Routing

| Working in | Read |
|---|---|
| Python — docstrings, module headers, LangGraph nodes | `references/python.md` |
| Node or TypeScript — TSDoc, exported types | `references/nodejs.md` |
| Go — GoDoc, package comments | `references/golang.md` |
| Checking what is undocumented | `scripts/docgate.py`, below |

## The four things

Every source file carries these four, in this order of importance. Anything
beyond them is optional and usually noise.

**1. A file header — what this file contains and which layer it is.**
One or two lines. Name the layer (`transport`, `service`, `repository`, `agent
node`, `schema`), because the layer is what tells the reader which rules apply
to the code inside.

**2. A one-line summary on every class.** What it is responsible for, in a
sentence. If the class holds state across calls, say what the state is and when
it is safe to reuse the instance. Lifecycle is the part people get wrong.

**3. A one-line summary on every exported function.** Verb first, no restating
the signature. Then, only where they are not obvious from the types: what it
returns, what it raises, and any side effect that leaves the process — a write,
a send, a spend, a model call.

**4. A called-from note on every exported function and every agent node.**
Where the thing is invoked from. This is the line that saves the most time and
the one nobody writes, because you only know it by searching — so generate it:

```bash
python .claude/skills/aptlogica-documentation/scripts/docgate.py --callers internal/services/auth/user.go
```

Format, in every language:

```
Called from: api.leads.create_lead, agents.outbound.draft_node
```

Three call sites maximum, then `and N others`. If the honest answer is
"nothing", write `Called from: nothing (dead code?)` — an export with no callers
is a finding, and writing it down is how it gets found.

## Worked example

The shape, in Python. The same four things in TSDoc and GoDoc are in the stack
references.

```python
"""Lead persistence. Repository layer — owns no transaction, returns domain
objects, never ORM rows.

Called from: services.leads, services.enrichment.
"""


class LeadRepository:
    """Reads and writes leads for one org. Holds a session, so one instance per
    request or per graph node — never a module-level singleton."""

    async def list_for_org(
        self, org_id: UUID, cursor: str | None, limit: int
    ) -> Page[Lead]:
        """Return one page of an org's leads, newest first.

        Cursor-based: offset pagination over this table skips rows while writes
        land. Raises `InvalidCursor` when the cursor does not decode.

        Called from: api.leads.list_leads, agents.outbound.select_targets.
        """
```

What that example does *not* contain is as important: no `:param org_id: the
org id`, no `:returns: a page of leads`, no restatement of the type hints. The
signature already says those. The docstring says the things the signature
cannot — the ordering guarantee, why the cursor exists, the error, the callers.

## What not to document

- **Anything the signature already states.** A parameter list that repeats the
  types is a second copy of the truth that rots at a different rate than the
  first.
- **Private helpers under about five lines.** Name them well instead.
- **Getters, setters, `__repr__`, trivial constructors.**
- **History.** "Changed in v2 to handle nulls" belongs in the commit, not the
  file. Nobody can tell whether an in-code history note is still true.
- **Commented-out code.** Delete it; git has it.

## Staleness

A wrong docstring is worse than no docstring, because it is read and believed.

- If a change moves behaviour, the docstring on that function moves in the same
  commit. This is not a follow-up task.
- If a function's callers change, its `Called from:` line changes with them.
  `docgate.py --callers` regenerates it; do not hand-maintain the list.
- A docstring that contradicts the code is a `medium` in review, and a `high`
  when it describes a security or approval behaviour the code no longer has.

## Documenting agent nodes

A node gets everything above plus three lines that exist nowhere else, because
this is the only place a reader can see the blast radius without tracing the
graph:

```
Tools:   send_email, read_crm            (mirrors ALLOWED_TOOLS)
Budget:  40_000 tokens, ceiling 120_000  (mirrors TOKEN_BUDGET, CONTEXT_CEILING)
Emits:   planning → calling_tool → done  (the states this node can emit)
```

Mirror the declared values; do not restate them from memory. If the docstring
and the declaration disagree, the declaration wins and the docstring was a bug.

## The generator

`scripts/docgate.py` finds what is undocumented. It is standard library only and
takes the same exit codes as the engineering gate — `0` clean, `1` findings,
`2` could not run. Like `gate.py`, the canonical copy lives in this skill and a
repository gets it by copying:

```bash
cp .claude/skills/aptlogica-documentation/scripts/docgate.py scripts/docgate.py
```

```bash
python scripts/docgate.py                      # whole repo, human-readable
python scripts/docgate.py --stack python        # one stack
python scripts/docgate.py --report reports/docs.json
python scripts/docgate.py --callers internal/services/auth/user.go   # regenerate Called-from lines
python scripts/docgate.py --paths src/agents    # limit the scan
```

What it checks: a module or package header on every source file; a summary on
every exported class, function and method; a `Called from:` line on every
module-level exported function in a service, repository, transport or agent
directory; and the three node lines on anything under an agent directory.

Methods are deliberately exempt from the `Called from:` check — on a repository
class every method would carry near-identical callers, and a line that is always
the same stops being read. Write it on the method anyway where the callers are
genuinely interesting; the file header carries the rest.

What it cannot check is whether the sentence is true. That is review's job — it
is item 8 in `aptlogica-review`.

Wire it next to the quality gate so documentation fails the same way everything
else does:

```bash
python scripts/gate.py --report reports/gate.json && \
python scripts/docgate.py --report reports/docs.json
```

## Writing docs for an existing file

When asked to document code that already exists:

1. Read the whole file first. A summary written from the top half of a file is
   the most common way a wrong docstring gets committed.
2. Run `docgate.py --callers <file>` before writing any `Called from:` line.
   Never guess the callers — a guessed caller list is read as fact.
3. Write the file header, then classes, then exported functions. Stop there;
   documenting private helpers is how a documentation pass turns into a diff
   nobody can review.
4. Re-run `docgate.py` on the file and paste the exit code.
5. Change no behaviour in the same commit. A documentation commit that also
   fixes a bug hides the bug fix from review.
