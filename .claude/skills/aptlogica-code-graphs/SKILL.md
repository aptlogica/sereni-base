---
name: aptlogica-code-graphs
description: "Build and query a persistent knowledge graph over a folder of code, docs, papers, images or video — Aptlogica's packaging of the graphify tool, which it still runs under the hood. Use when the user runs /aptlogica-code-graphs or /graphify, asks to graph or map a codebase or corpus, asks for a code graph or an architecture map, or asks a question about a codebase whose graphify-out/graph.json already exists — there, treat the question as a graphify query before reading files by hand. Not the general fallback for code questions: for writing, reviewing or documenting code in an Aptlogica repository use aptlogica-engineering, aptlogica-review or aptlogica-documentation."
version: 0.9.58
role: Knowledge Graph
allowed-tools: Read, Write, Edit, Bash, Glob, Grep, Agent, Task
---

<!--
Local fork, renamed. Upstream: https://github.com/safishamsi/graphify
(pip: graphifyy), Apache-2.0 — LICENSE and NOTICE sit beside this file and the
list below is the modification notice §4 requires.

`version` tracks the upstream package version this file was aligned to; it is
also in `.graphify_version`. Local changes made on top of upstream 0.9.58:

  * Step 2 passes `extra_excludes=['.claude', 'graphify-out', 'reports']` to
    `detect()`, so a repository with skills installed no longer graphs its own
    tooling — upstream never used the argument, though the API has it
  * renamed to `aptlogica-code-graphs`, so it sits with the other Aptlogica
    skills in the catalog. Only the skill name and the slash command moved:
    the `graphify` CLI, the `graphify.*` Python modules, `graphify-out/` and
    every sidecar keep their names, and `/graphify` still reaches this skill
    through the description

  * corpus path is read from `graphify-out/.graphify_root` as `_ROOT` instead
    of being hand-substituted into every Python block (paths with spaces or
    quotes silently built a graph on the wrong root)
  * `--directed`, the community labels and the extraction-prompt path are read
    from sidecar files too, so no Python block has a placeholder substituted
    into it any more (`IS_DIRECTED`, `LABELS_DICT`, `SPEC_PATH` are gone)
  * Step 3 and Step 9 moved to `references/extraction.md` and
    `references/finalise.md`; SKILL.md is 451 lines instead of 756
  * inline issue numbers removed — they were changelog, loaded as context on
    every invocation
  * `--force` and `--no-cluster` documented in Usage — error text told users to
    run flags that `--help` never printed
  * description narrowed so it stops competing with the Aptlogica skills for
    every code question
  * frontmatter gained `version` and `allowed-tools` (both `Agent` and
    `Task` are listed: Step B2 dispatches subagents, and the host's name
    for that tool differs by version — dropping the wrong one silently
    disables parallel semantic extraction)

Known upstream behaviour, not fixed here:

  * the AST extractor emits `imports` edges to stdlib modules it never creates
    nodes for, so Step 4.5 reports dangling endpoints on any Python corpus
    (25 of them on a 27-file repo: argparse, json, re, ...). Harmless, but the
    health check cries wolf and a real dangling edge is easy to miss in the noise
  * Step 4.5 counts AST multi-relation pairs (`imports_from` + `re_exports`
    between the same two modules) as collapsed edges, for the same reason

Re-apply these after a `graphifyy` upgrade overwrites this file, or drop the
fork and upstream them.
-->

# /aptlogica-code-graphs

Turn any folder of files into a navigable knowledge graph with community detection, an honest audit trail, and three outputs: interactive HTML, GraphRAG-ready JSON, and a plain-language GRAPH_REPORT.md.

## Usage

```
/aptlogica-code-graphs                                             # full pipeline on current directory (HTML viz; add --obsidian for a vault)
/aptlogica-code-graphs <path>                                      # full pipeline on specific path
/aptlogica-code-graphs https://github.com/<owner>/<repo>           # clone repo then run full pipeline on it
/aptlogica-code-graphs https://github.com/<owner>/<repo> --branch <branch>  # clone a specific branch
/aptlogica-code-graphs <url1> <url2> ...                           # clone multiple repos, build each, merge into one cross-repo graph
/aptlogica-code-graphs <path> --mode deep                          # thorough extraction, richer INFERRED edges
/aptlogica-code-graphs <path> --update                             # incremental - re-extract only new/changed files
/aptlogica-code-graphs <path> --directed                            # build directed graph (preserves edge direction: source→target)
/aptlogica-code-graphs <path> --whisper-model medium                # use a larger Whisper model for better transcription accuracy
/aptlogica-code-graphs <path> --cluster-only                       # rerun clustering on existing graph
/aptlogica-code-graphs <path> --force                              # rebuild even when the new graph has fewer nodes (runs `graphify extract <path> --force`)
/aptlogica-code-graphs <path> --no-cluster                         # skip community detection on a large flat corpus (runs `graphify extract <path> --no-cluster`)
/aptlogica-code-graphs <path> --no-viz                             # skip visualization, just report + JSON
/aptlogica-code-graphs <path> --html                               # (HTML is generated by default - this flag is a no-op)
/aptlogica-code-graphs <path> --svg                                # also export graph.svg (embeds in Notion, GitHub)
/aptlogica-code-graphs <path> --graphml                            # export graph.graphml (Gephi, yEd)
/aptlogica-code-graphs <path> --neo4j                              # generate graphify-out/cypher.txt for Neo4j
/aptlogica-code-graphs <path> --neo4j-push bolt://localhost:7687   # push directly to Neo4j
/aptlogica-code-graphs <path> --falkordb                           # generate graphify-out/cypher.txt for FalkorDB
/aptlogica-code-graphs <path> --falkordb-push falkordb://localhost:6379   # push directly to FalkorDB
/aptlogica-code-graphs <path> --mcp                                # start MCP stdio server for agent access
/aptlogica-code-graphs <path> --watch                              # watch folder, auto-rebuild on code changes (no LLM needed)
/aptlogica-code-graphs <path> --wiki                               # build agent-crawlable wiki (index.md + one article per community)
/aptlogica-code-graphs <path> --obsidian --obsidian-dir ~/vaults/my-project  # write vault to custom path (e.g. existing vault)
/aptlogica-code-graphs add <url>                                   # fetch URL, save to ./raw, update graph
/aptlogica-code-graphs add <url> --author "Name"                   # tag who wrote it
/aptlogica-code-graphs add <url> --contributor "Name"              # tag who added it to the corpus
/aptlogica-code-graphs query "<question>"                          # BFS traversal - broad context
/aptlogica-code-graphs query "<question>" --dfs                    # DFS - trace a specific path
/aptlogica-code-graphs query "<question>" --budget 1500            # cap answer at N tokens
/aptlogica-code-graphs path "AuthModule" "Database"                # shortest path between two concepts
/aptlogica-code-graphs explain "SwinTransformer"                   # plain-language explanation of a node
```

## What graphify is for

Drop any folder of code, docs, papers, images, or video into graphify and get a queryable knowledge graph. Persistent across sessions, honest audit trail (EXTRACTED/INFERRED/AMBIGUOUS), community detection surfaces cross-document connections you wouldn't think to ask about.

## What You Must Do When Invoked

If the user invoked `/aptlogica-code-graphs --help` or `/aptlogica-code-graphs -h` (with no other arguments), print the contents of the `## Usage` section above verbatim and stop. Do not run any commands, do not detect files, do not default the path to `.`. Just print the Usage block and return.

**Fast path — existing graph:** Before doing anything else, check whether `graphify-out/graph.json` exists. The expected location is `graphify-out/graph.json` relative to the **current working directory** (i.e. the project root where you are running commands). If it exists AND the user's request is a natural-language question about the codebase (e.g. "How does X work?", "What calls Y?", "Trace the data flow through Z") and NOT an explicit rebuild command (`--update`, `--cluster-only`, or a bare path/URL that implies fresh extraction): **skip Steps 1–5 entirely and jump straight to `## For /aptlogica-code-graphs query`.** Run `graphify query "<question>"` immediately. Do not run detect. Do not check corpus size. Do not ask the user to narrow. The graph is already built — use it.

If no path was given, use `.` (current directory). Do not ask the user for a path.

If the path argument starts with `https://github.com/` or `http://github.com/`, treat it as a GitHub URL - run Step 0 before anything else, then continue with the resolved local path.

Follow these steps in order. Do not skip steps.

### Step 0 - GitHub repos and multi-path merge (only if a URL or several paths)

Only when the path is one or more `https://github.com/...` URLs, or several local subfolders to merge. See `references/github-and-merge.md` for the clone, cross-repo merge, and monorepo flow, then continue with the resolved local path. A plain local path skips this step.

### Step 1 - Ensure graphify is installed

```bash
# Detect the correct Python interpreter (handles uv tool, pipx, venv, system installs)
PYTHON=""
GRAPHIFY_BIN=$(which graphify 2>/dev/null)
# 1. uv tool installs — most reliable on modern Mac/Linux
if [ -z "$PYTHON" ] && command -v uv >/dev/null 2>&1; then
    _UV_PY=$(uv tool run --from graphifyy python -c "import sys; print(sys.executable)" 2>/dev/null)
    if [ -n "$_UV_PY" ]; then PYTHON="$_UV_PY"; fi
fi
# 2. Read shebang from graphify binary (pipx and direct pip installs)
if [ -z "$PYTHON" ] && [ -n "$GRAPHIFY_BIN" ]; then
    _SHEBANG=$(head -1 "$GRAPHIFY_BIN" | tr -d '#!')
    case "$_SHEBANG" in
        *[!a-zA-Z0-9/_.@-]*) ;;
        *) "$_SHEBANG" -c "import graphify" 2>/dev/null && PYTHON="$_SHEBANG" ;;
    esac
fi
# 3. Fall back to python3
if [ -z "$PYTHON" ]; then PYTHON="python3"; fi
if ! "$PYTHON" -c "import graphify" 2>/dev/null; then
    if command -v uv >/dev/null 2>&1; then
        uv tool install --upgrade graphifyy -q 2>&1 | tail -3
        _UV_PY=$(uv tool run --from graphifyy python -c "import sys; print(sys.executable)" 2>/dev/null)
        if [ -n "$_UV_PY" ]; then PYTHON="$_UV_PY"; fi
    else
        "$PYTHON" -m pip install graphifyy -q 2>/dev/null \
          || "$PYTHON" -m pip install graphifyy -q --break-system-packages 2>&1 | tail -3
    fi
fi
# Write interpreter path for all subsequent steps (persists across invocations)
mkdir -p graphify-out
"$PYTHON" -c "import sys; open('graphify-out/.graphify_python', 'w', encoding='utf-8').write(sys.executable)"
# Save scan root so `graphify update` (no args) knows where to look next time
echo "$(cd "INPUT_PATH" && pwd)" > graphify-out/.graphify_root
# Record the graph direction once, here, rather than substituting True/False
# into every later Python block. Write 1 if --directed was given, else 0.
echo "0" > graphify-out/.graphify_directed
```

If the import succeeds, print nothing and move straight to Step 2.

**In every subsequent bash block, replace `python3` with `$(cat graphify-out/.graphify_python)` to use the correct interpreter.**

### Step 2 - Detect files

```bash
$(cat graphify-out/.graphify_python) -c "
import json
from graphify.detect import detect
from pathlib import Path
_ROOT = Path('graphify-out/.graphify_root').read_text(encoding='utf-8').strip()
# Agent tooling is not the project. Without this, a repository with skills
# installed graphs its own standard: on aptl-vantage that was 30 of 36
# documents. detect() has always taken extra_excludes; graphify's own runbook
# never used it.
result = detect(Path(_ROOT), extra_excludes=['.claude', 'graphify-out', 'reports'])
# Write the sidecar from Python, not a shell redirect, so the same block renders
# on PowerShell hosts without console-encoding drift.
Path('graphify-out/.graphify_detect.json').write_text(json.dumps(result, ensure_ascii=False), encoding=\"utf-8\")
print(f'Detected {result[\"total_files\"]} files')
"
```

The path is not substituted here: Step 1 already resolved it to an absolute
path in `graphify-out/.graphify_root`, and every later block reads it from
there as `_ROOT`. That is deliberate — a hand-substituted path breaks on a
space, an apostrophe or a backslash, and the break surfaces as a graph built
on the wrong root rather than as an error. Step 1 is the only place the raw
path appears, and it is quoted.

Do NOT cat or print the JSON - read it silently and present a clean summary instead:

```
Corpus: X files · ~Y words
  code:     N files (.py .ts .go ...)
  docs:     N files (.md .txt ...)
  papers:   N files (.pdf ...)
  images:   N files
  video:    N files (.mp4 .mp3 ...)
```

Omit any category with 0 files from the summary.

Then act on it:
- If `total_files` is 0: stop with "No supported files found in [path]."
- If `skipped_sensitive` is non-empty: report the count and list the skipped file names, so a wrongly-flagged source or doc is visible and can be renamed or moved.
- If `total_words` > 2,000,000 OR `total_files` > 500: show the warning. Then compute the top 5 first-level subdirectories by file count:
  - Read `scan_root` from the detect JSON (always an absolute path to the resolved corpus root).
  - Concatenate all file lists across all types (`code`, `document`, `paper`, `image`, `video`).
  - Filter out any path that starts with `scan_root + "/graphify-out/"` to exclude converted sidecars.
  - For each file, strip the `scan_root` prefix and take the first path component. Files directly in `scan_root` with no subdirectory count as `(root)`.
  - If all files are in `(root)` with no subdirectories, do not ask to narrow — no subfolders exist. Instead suggest `--no-cluster` (it runs `graphify extract <path> --no-cluster`) to skip the expensive clustering step, and proceed.
  - Otherwise rank by count, show the top 5 with file counts, then ask which subfolder to run on. Wait for the user's answer before proceeding.
- Otherwise: proceed directly to Step 2.5 if video files were detected, or Step 3 if not.

### Step 2.5 - Video and audio (only if video files detected)

Skip this step entirely if `detect` returned zero `video` files. When the corpus has video or audio, see `references/transcribe.md` to transcribe them to text first, then treat the transcripts as doc files in Step 3.

### Step 3 - Extract entities and relationships

Two parts, run in parallel in one message: **Part A** structural extraction from
code (AST, deterministic, free) and **Part B** semantic extraction from docs,
papers and images (LLM). **Part C** merges them.

> **graphify needs no API key. Never ask the user for one, and never block on
> one.** Code is extracted structurally with no LLM at all, so the common
> `/aptlogica-code-graphs .` on a repository skips Part B entirely. Semantic extraction uses
> Gemini only if `GEMINI_API_KEY`/`GOOGLE_API_KEY` is already set; otherwise the
> host agent itself is the LLM. graphify never reads `ANTHROPIC_API_KEY` or any
> other provider key.

The runbook — the AST block, the cache check, subagent dispatch and chunking,
the merge, and the exact commands — is in `references/extraction.md`. Read it
now; it is the longest step and the one with the most ways to go wrong.

### Step 4 - Build graph, cluster, analyze, generate outputs

**Before starting:** nothing is substituted into the blocks below. The corpus path is read from `graphify-out/.graphify_root` as `_ROOT`, and the graph direction from `graphify-out/.graphify_directed` as `_DIRECTED` — Step 1 wrote `0`; if `--directed` was given, write `1` to that file now (`echo 1 > graphify-out/.graphify_directed`) before running anything here. A directed build preserves edge direction source→target; the default is an undirected `Graph`.

The corpus path needs no substitution anywhere below: `_ROOT` is read from `graphify-out/.graphify_root`.

```bash
mkdir -p graphify-out
$(cat graphify-out/.graphify_python) -c "
import sys, json
from graphify.build import build_from_json
from graphify.cluster import cluster, score_all
from graphify.analyze import god_nodes, surprising_connections, suggest_questions
from graphify.report import generate
from graphify.export import to_json
from pathlib import Path
_ROOT = Path('graphify-out/.graphify_root').read_text(encoding='utf-8').strip()
_DIRECTED = Path('graphify-out/.graphify_directed').read_text(encoding='utf-8').strip() == '1'

extraction = json.loads(Path('graphify-out/.graphify_extract.json').read_text(encoding=\"utf-8\"))
detection  = json.loads(Path('graphify-out/.graphify_detect.json').read_text(encoding=\"utf-8\"))

# root= mirrors the --update runbook: relativize source_file to the same
# base so the full build and incremental --update never drift apart on re-extract.
G = build_from_json(extraction, root=_ROOT, directed=_DIRECTED)
# Guard BEFORE any write: an empty extraction must not clobber a good graph.json /
# GRAPH_REPORT.md / analysis sidecar. Check immediately after build.
if G.number_of_nodes() == 0:
    print('ERROR: Graph is empty - extraction produced no nodes.')
    print('Possible causes: all files were skipped, binary-only corpus, or extraction failed.')
    raise SystemExit(1)
communities = cluster(G)
cohesion = score_all(G, communities)
tokens = {'input': extraction.get('input_tokens', 0), 'output': extraction.get('output_tokens', 0)}
gods = god_nodes(G)
surprises = surprising_connections(G, communities)
labels = {cid: 'Community ' + str(cid) for cid in communities}
# Placeholder questions - regenerated with real labels in Step 5
questions = suggest_questions(G, communities, labels)

# Export FIRST and honor the shrink-guard: to_json returns False (writing
# nothing) when the new graph is smaller than the existing graph.json. Only write
# GRAPH_REPORT.md + the analysis sidecar when the graph was actually written, so
# they never describe a graph that graph.json doesn't contain.
wrote = to_json(G, communities, 'graphify-out/graph.json')
if not wrote:
    print('ERROR: refused to shrink graphify-out/graph.json (existing graph has more nodes).')
    print('If this shrink is intentional (you deleted files), re-run: graphify extract <path> --force')
    raise SystemExit(1)
report = generate(G, communities, cohesion, labels, gods, surprises, detection, tokens, _ROOT, suggested_questions=questions)
Path('graphify-out/GRAPH_REPORT.md').write_text(report, encoding=\"utf-8\")
analysis = {
    'communities': {str(k): v for k, v in communities.items()},
    'cohesion': {str(k): v for k, v in cohesion.items()},
    'gods': gods,
    'surprises': surprises,
    'questions': questions,
}
Path('graphify-out/.graphify_analysis.json').write_text(json.dumps(analysis, indent=2, ensure_ascii=False), encoding=\"utf-8\")
print(f'Graph: {G.number_of_nodes()} nodes, {G.number_of_edges()} edges, {len(communities)} communities')
"
```

If this step prints `ERROR: Graph is empty`, stop and tell the user what happened - do not proceed to labeling or visualization.


### Step 4.5 - Graph health check (read-only integrity gate)

A non-destructive diagnostic on the extraction, before labeling. It surfaces edge collapse, dangling/missing endpoints, and self-loops — the silent-corruption modes of incremental updates and AST/LLM id mismatches. Read-only; never aborts.

```bash
$(cat graphify-out/.graphify_python) -c "
import json
from pathlib import Path
_ROOT = Path('graphify-out/.graphify_root').read_text(encoding='utf-8').strip()
_DIRECTED = Path('graphify-out/.graphify_directed').read_text(encoding='utf-8').strip() == '1'
from graphify.diagnostics import diagnose_extraction, format_diagnostic_report

extraction = json.loads(Path('graphify-out/.graphify_extract.json').read_text(encoding=\"utf-8\"))
summary = diagnose_extraction(extraction, directed=_DIRECTED, root=_ROOT)
print(format_diagnostic_report(summary))
flags = [f'{summary[k]} {label}' for k, label in (
    ('dangling_endpoint_edges', 'dangling-endpoint edges'),
    ('missing_endpoint_edges', 'missing-endpoint edges'),
    ('self_loop_edges', 'self-loop edges'),
    ('directed_same_endpoint_collapsed_edges', 'collapsed (directed) edges'),
    ('undirected_same_endpoint_collapsed_edges', 'collapsed (undirected) edges'),
) if summary.get(k, 0)]
print('GRAPH HEALTH WARNING: ' + '; '.join(flags) + ' - graph may be incomplete/corrupt.' if flags else 'Graph health: OK (no dangling/missing/collapsed edges).')
"
```

If a `GRAPH HEALTH WARNING` prints, surface it in the final summary (do not abort — the graph is still usable, but the integrity issue must be visible, per the Honesty Rules).

### Step 5 - Label communities

Read `graphify-out/.graphify_analysis.json`. For each community key, look at its node labels and write a 2-5 word plain-language name (e.g. "Attention Mechanism", "Training Pipeline", "Data Loading").

Then regenerate the report and save the labels for the visualizer:

```bash
$(cat graphify-out/.graphify_python) -c "
import sys, json
from graphify.build import build_from_json
from graphify.cluster import score_all
from graphify.analyze import god_nodes, surprising_connections, suggest_questions
from graphify.report import generate
from graphify.export import to_json
from pathlib import Path
_ROOT = Path('graphify-out/.graphify_root').read_text(encoding='utf-8').strip()
_DIRECTED = Path('graphify-out/.graphify_directed').read_text(encoding='utf-8').strip() == '1'

extraction = json.loads(Path('graphify-out/.graphify_extract.json').read_text(encoding=\"utf-8\"))
detection  = json.loads(Path('graphify-out/.graphify_detect.json').read_text(encoding=\"utf-8\"))
analysis   = json.loads(Path('graphify-out/.graphify_analysis.json').read_text(encoding=\"utf-8\"))

# root= as in Step 4 / the --update runbook — same base for node-key parity.
G = build_from_json(extraction, root=_ROOT, directed=_DIRECTED)
communities = {int(k): v for k, v in analysis['communities'].items()}
cohesion = {int(k): v for k, v in analysis['cohesion'].items()}
tokens = {'input': extraction.get('input_tokens', 0), 'output': extraction.get('output_tokens', 0)}

# Labels are read from the file you wrote above, so no dict is substituted into
# this block and a label containing a quote cannot break it.
labels = {
    int(k): v
    for k, v in json.loads(
        Path('graphify-out/.graphify_labels_input.json').read_text(encoding=\"utf-8\")
    ).items()
}

# Regenerate questions with real community labels (labels affect question phrasing)
questions = suggest_questions(G, communities, labels)

report = generate(G, communities, cohesion, labels, analysis['gods'], analysis['surprises'], detection, tokens, _ROOT, suggested_questions=questions)
Path('graphify-out/GRAPH_REPORT.md').write_text(report, encoding=\"utf-8\")
Path('graphify-out/.graphify_labels.json').write_text(json.dumps({str(k): v for k, v in labels.items()}, ensure_ascii=False), encoding=\"utf-8\")
# Re-export so graph.json nodes carry the curated community_name.
# Same extraction as Step 4, so the  shrink-guard passes on node count;
# if it still refuses, surface the guard message - do not force past it.
wrote = to_json(G, communities, 'graphify-out/graph.json', community_labels=labels)
if not wrote:
    print('ERROR: refused to shrink graphify-out/graph.json (existing graph has more nodes).')
    print('If this shrink is intentional (you deleted files), re-run: graphify extract <path> --force')
print('Report updated with community labels')
"
```

Before running that block, write the names you chose to
`graphify-out/.graphify_labels_input.json` as a JSON object keyed by community
id, for example `{"0": "Attention Mechanism", "1": "Training Pipeline"}`. The
block reads that file, so a label containing a quote or a brace cannot break the
code.

### Step 6 - Generate Obsidian vault (opt-in) + HTML

**Generate HTML always** (unless `--no-viz`). **Obsidian vault only if `--obsidian` was explicitly given** — skip it otherwise, it generates one file per node.

If `--obsidian` was given:

- If `--obsidian-dir <path>` was also given, pass it via `--dir`. Otherwise defaults to `graphify-out/obsidian`.

```bash
graphify export obsidian
# or with custom dir: graphify export obsidian --dir ~/vaults/my-project
```

Generate the HTML graph (always, unless `--no-viz`):

```bash
graphify export html  # auto-aggregates to community view if graph > 5000 nodes
# or: graphify export html --no-viz
```

### Steps 6b-8 - Wiki, Neo4j, FalkorDB, SVG, GraphML, MCP, benchmark (only on their flags)

These run only when their flag is present (`--wiki`, `--neo4j`/`--neo4j-push`, `--falkordb`/`--falkordb-push`, `--svg`, `--graphml`, `--mcp`) or, for the token-reduction benchmark, when `total_words` exceeds 5,000. A default run with no export flags skips all of them. See `references/exports.md` for each one. Run any `--wiki` export before Step 9 cleanup so `.graphify_labels.json` is still available.

---

### Step 9 - Save manifest, update cost tracker, clean up, and report

Writes the manifest `--update` reads next time, appends this run to
`graphify-out/cost.json`, deletes the sidecars, then prints the closing summary
and offers to trace the most interesting suggested question.

Only files that actually produced output are stamped in the manifest: a detected
file whose chunk failed must stay unstamped so the next `--update` re-queues it.
The commands are in `references/finalise.md`.

---

## Interpreter guard for subcommands

Before running any subcommand below (`--update`, `--cluster-only`, `query`, `path`, `explain`, `add`), check that `.graphify_python` exists. If it's missing (e.g. user deleted `graphify-out/`), re-resolve the interpreter first:

```bash
if [ ! -f graphify-out/.graphify_python ]; then
    GRAPHIFY_BIN=$(which graphify 2>/dev/null)
    if [ -n "$GRAPHIFY_BIN" ]; then
        PYTHON=$(head -1 "$GRAPHIFY_BIN" | tr -d '#!')
        case "$PYTHON" in *[!a-zA-Z0-9/_.@-]*) PYTHON="python3" ;; esac
    else
        PYTHON="python3"
    fi
    mkdir -p graphify-out
    "$PYTHON" -c "import sys; open('graphify-out/.graphify_python', 'w', encoding='utf-8').write(sys.executable)"
fi
```

## For --update and --cluster-only

Both are non-default subcommands. `--update` re-extracts only new or changed files; `--cluster-only` reruns clustering on the existing graph. See `references/update.md` for both flows.

---

## For /aptlogica-code-graphs query

When `graphify-out/graph.json` already exists and the user asks a question about the corpus, answer from the graph rather than rebuilding it:

```bash
graphify query "<question>"
```

Before traversal, expand the question against the graph's own vocabulary so a wording mismatch does not collapse the answer to noise. If the `graphify query` CLI is unavailable, fall back to an inline NetworkX traversal of `graphify-out/graph.json`. Answer using only what the graph output contains, and quote `source_location` when citing a specific fact. For that vocab-expansion step, the BFS/DFS traversal modes, the `--budget` cap, the NetworkX fallback, `save-result` feedback, and the `/aptlogica-code-graphs path` and `/aptlogica-code-graphs explain` flows, see `references/query.md`.

---

## For /aptlogica-code-graphs add and --watch

Neither is part of the default build. When the user runs `/aptlogica-code-graphs add <url>` to fetch a URL into the corpus, or passes `--watch` to auto-rebuild on file changes, see `references/add-watch.md`.

---

## For the commit hook and native CLAUDE.md integration

When the user asks to install the post-commit auto-rebuild hook or wire graphify into a project's CLAUDE.md, see `references/hooks.md`.

---

## Honesty Rules

- Never invent an edge. If unsure, use AMBIGUOUS.
- Never skip the corpus check warning.
- Always show token cost in the report.
- Never hide cohesion scores behind symbols - show the raw number.
- Never run HTML viz on a graph with more than 5,000 nodes without warning the user.
