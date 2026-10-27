# Eval prompts — aptlogica-code-graphs (renamed graphify fork)

Run after re-applying the fork changes on a `graphifyy` upgrade. Cases 1–4 guard
the description narrowing; 5–9 guard the local edits that upstream does not have.

## Selection

| # | Prompt | Pass condition |
|---|---|---|
| 1 | `/aptlogica-code-graphs` | Fires; runs the pipeline on the current directory |
| 2 | "where is `create_task` defined?" in a repo with an index | `aptlogica-memory` fires, not this skill |
| 3 | "how do these three subsystems connect?" with `graphify-out/` present | Fires; treats it as a query, does not rebuild |
| 4 | "review my diff" | Does **not** fire |

## Behaviour — the fork's own changes

| # | Situation | Pass condition |
|---|---|---|
| 5 | Corpus path contains a space or an apostrophe | Build succeeds; no block hand-substitutes the path — every step reads `_ROOT` from `graphify-out/.graphify_root` |
| 6 | `--directed` given | `.graphify_directed` written in Step 1 and read by Steps 4, 4.5 and 5; no literal `IS_DIRECTED` reaches Python |
| 7 | Community labelling | Labels written to `.graphify_labels_input.json` and read back; no `LABELS_DICT` literal in the executed block |
| 8 | `/aptlogica-code-graphs --help` | Prints the Usage block verbatim, including `--force` and `--no-cluster` |
| 9 | Graph would shrink | Error names the real command (`graphify extract <path> --force`), not a flag that does not exist at that level |

## Negative

| # | Prompt | Pass condition |
|---|---|---|
| 10 | No API key set, code-only corpus | Proceeds; never prompts for `ANTHROPIC_API_KEY` |
| 11 | Corpus of 4 files | Builds without asking the user to narrow the scope |
