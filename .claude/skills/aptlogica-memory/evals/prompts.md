# Eval prompts — aptlogica-memory

Run after any change to `SKILL.md` or `scripts/memory.py`. Cases 5–8 are the
ones that matter: they are the difference between an index that saves tokens and
one that produces confident wrong answers.

## Selection

| # | Prompt | Pass condition |
|---|---|---|
| 1 | "where is `create_task` defined?" | `memory.py --query` before any grep or file read |
| 2 | "what calls `humanAcceptance`?" | `--query`; reports the caller list as textual, not as proof |
| 3 | "build the repo index" / "the index is stale" | Runs the builder; commits `.claude/memory/` with the change |
| 4 | "how does the outbound subsystem hang together?" | Says the index answers *where*, not *how*, and offers aptlogica-code-graphs |

## Behaviour

| # | Situation | Pass condition |
|---|---|---|
| 5 | `--query` returns nothing for a symbol | Runs `--verify` before concluding the symbol does not exist |
| 6 | Asked what a function *does*, entry in hand | Opens the file; does not answer from the index summary alone |
| 7 | `--verify` exits 1 mid-task | Rebuilds before trusting the next lookup |
| 8 | About to change an exported signature | Uses the caller list as a search starting point and says it may be incomplete |
| 9 | Asked to hand-edit `INDEX.md` | Refuses; edits the code and rebuilds |
| 10 | Merge conflict in `index.json` | Rebuilds rather than resolving by hand |
| 11 | Asked where a decision was recorded | Points at `DECISIONS.md` / `ARCHITECTURE.md`; does not invent an index entry for it |

## Negative

| # | Prompt | Pass condition |
|---|---|---|
| 12 | "run the tests" | Does not rebuild the index first |
| 13 | Single-file repo with no `src/` | Says there is nothing worth indexing rather than indexing `node_modules` |
