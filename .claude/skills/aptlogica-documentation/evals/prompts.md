# Eval prompts — aptlogica-documentation

Run after any change to `SKILL.md`, a stack reference, or `scripts/docgate.py`.

## Selection

| # | Prompt | Pass condition |
|---|---|---|
| 1 | "add docstrings to this file" | This skill fires |
| 2 | "what does this file contain?" | Fires, and answers in the file-header shape |
| 3 | "document the API" | Asks whether that means in-code docs or `docs/API_SPEC.md`; does not silently pick |
| 4 | "write a README for this service" | Does **not** claim this standard covers it — this skill is in-code documentation |

## Behaviour

| # | Situation | Pass condition |
|---|---|---|
| 5 | Documenting `list_for_org(org_id, cursor, limit) -> Page[Lead]` | No `:param` restating types; states ordering, the cursor rationale, the raise |
| 6 | Asked for a `Called from:` line | Runs `docgate.py --callers` first; never guesses the callers |
| 7 | An export with no callers | Writes `Called from: nothing (dead code?)` rather than omitting the line |
| 8 | Documenting an agent node | Adds Tools/Budget/Emits, mirroring the declarations rather than restating from memory |
| 9 | A docstring contradicts the code it sits on | Treats it as a defect, fixes the docstring, and does not change behaviour in the same commit |
| 10 | Python file with `_helper()` three lines long | Leaves it undocumented; suggests a better name if the name is poor |
| 11 | `export type X = z.infer<typeof x>` | Documents the schema, not the alias |
| 12 | Asked to document a 900-line file | Reads the whole file before writing any summary, or says it has not |

## Negative

| # | Prompt | Pass condition |
|---|---|---|
| 13 | "rename this variable" | Makes the edit; does not open a documentation pass |
| 14 | "add a comment explaining this regex" | One comment; does not add headers to the whole file |
