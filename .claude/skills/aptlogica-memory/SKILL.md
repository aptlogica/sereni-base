---
name: aptlogica-memory
description: Aptlogica's repository memory — a deterministic index of every file, its layer, its exported symbols with line numbers, and who calls them, built by scripts/memory.py with no model involved. Use this skill to answer "where does X live", "what calls Y", "what is in this file" or "what exists already" before searching or reading the tree, to build or refresh the index, when the index is stale, and whenever an answer needs a file:line citation rather than a recollection. Also use when asked to reduce token usage or stop an agent guessing at code it has not read.
version: 1.2.0
role: Repository Memory
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Repository Memory

Two things go wrong on a long session, and they are the same thing seen from
different ends.

The expensive failure: the agent re-reads half the repository to remember where
something is, and the context that was holding the actual task gets squeezed out.

The dangerous failure: it does not re-read, and answers from memory of a file it
saw twenty turns ago — or never saw at all. That produces a confident claim
about code that does not exist, which is the development-time version of the
failure the whole standard is built to prevent.

`scripts/memory.py` addresses both with one artifact: a committed index of what
is in the repository, built by parsing, not by asking a model.

```bash
python scripts/memory.py            # build or refresh .claude/memory/
python scripts/memory.py --verify   # is it current? exit 1 if not
python scripts/memory.py --query create_task
python scripts/memory.py --file src/api/tasks.py   # one file's whole entry
```

```
src/api/tasks.py:17  create_task (fn) [transport]
    Create a task under an objective in scope. 422 without objective_id, 404 if unseen.
    callers (indexed, textual): workers.run_worker
```

The caller line says how it was found, because that is the part a reader must
not over-trust: it is a textual match, not a call graph.

That answer cost one command. The same answer by search-and-read costs several
thousand tokens and is one stale assumption away from being wrong.

## The three rules

**1. Index first, file second, never index only.**
Start from the index to find the thing. Then open the file before you assert
anything about what it *does*. The index carries names, lines, layers, callers
and the first line of the docstring — enough to navigate, never enough to
conclude. A summary is what the code said about itself at build time; behaviour
is what the code does now.

**2. Cite `file:line`, and get it from the index, not from memory.**
Every entry has a line number precisely so a citation is cheap. "The existing
handler already does this" is a claim needing evidence, and the index is where
the evidence starts.

**3. A stale index is a liability, so check it.**
Every entry stores the hash of the file it came from. `--verify` compares them
and exits 1 on drift, naming each file. If verify is red, rebuild before you
trust a lookup — a "not found" from a stale index is exactly how an agent
concludes something needs writing when it already exists.

## What it holds, and what it does not

Holds, per file: path, layer (`transport`, `service`, `repository`, `agent`,
`model`, `schema`, `core`, `test`, `other`), the header line, every exported
symbol with kind and line number, the first line of each symbol's docstring, a
textual caller list, and for agent nodes the declared `tools`, `budget`,
`ceiling` and `model`.

Does not hold: why anything is the way it is, what is planned, what was decided
and rejected, or any behaviour not stated in a docstring. Those live in
`docs/ARCHITECTURE.md`, `DEV_PLAN.md` and `DECISIONS.md`, and the index links to
them under **Contracts** rather than duplicating them — a second copy of a
decision is a decision that can silently disagree with itself.

The index is only as good as the docstrings, which is the other half of why the
documentation standard exists: `aptlogica-documentation` is what makes the
header lines worth reading.

## Caller lists are a starting point

Callers are found textually: a symbol's name appearing in another module counts
as a call. That over-reports on short or common names (which is why names under
four characters are skipped) and under-reports on dynamic dispatch, a
string-keyed registry, or a call through an interface.

So the caller list is where a search starts, never where a conclusion ends. If
you are about to change a signature, the index tells you which six modules to
open; it does not tell you that those are all of them.

## Where it fits in the development loop

`aptlogica-engineering` §6 runs the gate at the start and end of a task. Memory
sits beside it:

1. **Before starting** — `python scripts/memory.py --verify`. If it is stale,
   rebuild; that is a two-second command and it is the difference between
   navigating and guessing.
2. **While working** — `--query` instead of a repo-wide grep, for any "where is
   it / what calls it" question.
3. **Before committing** — rebuild. In SereniBase `.claude/memory/` is
   gitignored, so **do not commit it**: each developer builds it locally (see
   "Why it is committed" below).

Wire it next to the gates so it cannot rot unnoticed:

```bash
python scripts/gate.py --report reports/gate.json && \
python scripts/docgate.py --report reports/docs.json && \
python scripts/memory.py --verify
```

A pre-commit hook that runs `python scripts/memory.py` and stages
`.claude/memory/` keeps it current without anyone remembering to.

## Why it is committed

> **SereniBase:** not committed. This is a public repository with outside
> contributors, and a ~2 MB index that changes with almost every PR would make
> their PRs conflict constantly. `.claude/memory/` is in `.gitignore`. Build it
> with `python scripts/memory.py` and never stage it. The reasoning below
> applies to private Aptlogica repositories.

Because it is a shared starting point. An index each developer builds privately
helps that developer; an index in the repository means every agent session, in
every clone, on every machine, starts from the same map — and a drifted entry
shows up in a diff where someone can see it.

It is generated, so treat it like a lockfile: never hand-edit it, never resolve
a conflict in it by hand. Rebuild and commit the result.

## Relationship to aptlogica-code-graphs

Different tools for different questions, and it is worth being precise about
which, because running the wrong one is the expensive mistake:

- **memory.py** — "where does X live, what calls Y, what exists already".
  Deterministic, seconds, no model, no tokens. The default.
- **aptlogica-code-graphs** (graphify under the hood) — "how does this subsystem
  hang together, what connects these two
  areas, what would I never think to ask". Builds a knowledge graph, costs an
  extraction pass, and is worth it once per corpus, not once per task.

Use memory for navigation. Reach for the code graph when the question is about
structure across many files, and the index has already told you the shape is
bigger than a handful of modules.

## Getting it into a repository

```bash
cp .claude/skills/aptlogica-memory/scripts/memory.py scripts/memory.py
python scripts/memory.py
git add .claude/memory scripts/memory.py
```

Default roots are `src`, `internal`, `cmd`, `web/src`, `app` and `lib` — whichever
exist. Pass `--paths` for anything else. The index lands in `.claude/memory/`
as `INDEX.md` (one line per file, the part meant to be read) and `index.json`
(everything, the part `--query` reads).

Keep `INDEX.md` small enough to read whole. On a 137-file service it is about
19 KB, roughly 4k tokens — cheaper than opening four files, and it answers a
question that four files would not. If it grows past what you would willingly
load, index fewer roots rather than reading it less.
