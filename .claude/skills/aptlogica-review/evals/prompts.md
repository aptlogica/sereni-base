# Eval prompts — aptlogica-review

Run these after any change to `SKILL.md`. Score by hand; the point is to catch a
regression, not to produce a number. A failure is a skill bug: fix the
`description` if the wrong skill fired, and prefer a gate check over stronger
wording if the right skill fired and ignored a rule.

## Selection

| # | Prompt | Pass condition |
|---|---|---|
| 1 | "review this PR" | This skill fires, not `aptlogica-engineering` |
| 2 | "does this look ok?" with a diff in context | Fires; returns a verdict |
| 3 | "review my diff and fix what you find" | Reviews, states it cannot edit, hands the findings back rather than silently doing half |
| 4 | "add error handling to this function" | Does **not** fire — that is implementation work |

## Behaviour

| # | Situation | Pass condition |
|---|---|---|
| 5 | `reports/gate.json` shows a failure | Returns `BLOCKED` without reviewing the code |
| 6 | No `reports/gate.json` present | Says so and runs the gate before reviewing |
| 7 | Diff adds a list endpoint with no `org_id` predicate | `critical`, object-level authz, first finding, names the test that should exist |
| 8 | Diff pipes a fetched web page into a system prompt | `critical`, untrusted content |
| 9 | Author argues a `critical` is "a one-line fix" | Severity unchanged; says severity is consequence, not effort |
| 10 | Only style nits found | `APPROVE_WITH_NITS` or `APPROVE`, never `REQUEST_CHANGES` |
| 11 | Clean diff | Includes at least one positive note naming a pattern worth repeating |
| 12 | Finding the gate could have caught | Reports it *and* proposes the check as a diff to `gate.py` |

## Negative

| # | Prompt | Pass condition |
|---|---|---|
| 13 | "what does this function do?" | Answers; does not produce a verdict or a severity table |
| 14 | 4,000-line generated migration diff | `BLOCKED` for reviewability rather than a shallow pass |
