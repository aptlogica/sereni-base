---
name: aptlogica-review
description: Aptlogica's code review procedure — read-only review of a diff, branch, pull request or file, in the order that matters (object-level authorization, untrusted content paths, irreversible actions, correctness, contract conformance, performance, maintainability), with a severity scale and a required verdict. Use this skill whenever asked to review, audit, critique or sign off on code in the SereniBase repository — including casual phrasings ("does this look ok", "can you check my PR", "anything wrong with this diff", "review before I merge"). For writing or changing code, use aptlogica-engineering instead.
version: 3.2.0
role: Code Review
allowed-tools: Read, Grep, Glob, Bash
---

# Aptlogica Code Review

Review is read-only. This skill is granted no `Write` and no `Edit`, and that is
deliberate: a reviewer who fixes as they go produces a diff nobody reviewed and
destroys the record of what was wrong. If a finding is worth fixing, it is worth
the author fixing it with the finding visible next to it.

The standard being reviewed against is `aptlogica-engineering`. Read its
`SKILL.md` for the universal rules and the one `references/` file matching the
stack in the diff. This skill is only the procedure.

## Before reading any code

Read `reports/gate.json`. Do not re-review what the gate covers — formatting,
typing, secrets, known-vulnerable dependencies, traceability comments, node
contract declarations, coverage.

If the gate is red, stop and return `BLOCKED`. Reviewing code that has not
passed its own mechanical checks trains the team to treat the gate as optional.

If `reports/gate.json` is absent, say so and run `python scripts/gate.py
--report reports/gate.json` yourself before reviewing. An unreviewed gate result
is not the same as a green one.

## Review order

In this order, so running out of attention still leaves the important things
covered:

1. **Object-level authorization** on every changed endpoint and every node that
   reads data. Can a caller reach another org's records? The most common serious
   bug in a multi-tenant product and the least visible in a diff, because the
   code looks correct in isolation.
2. **Untrusted content paths** — is external text wrapped as untrusted, and
   could it reach an undeclared tool?
3. **Irreversible actions** — named-approver check, scoped to this action and
   this run?
4. **Correctness** — error paths, boundary inputs, transaction placement,
   concurrency, whether async work is actually awaited.
5. **Contract conformance** against `docs/API_SPEC.md` and the state schema.
6. **Performance** — N+1, unpaginated lists, a new query pattern with no index.
7. **Maintainability** — logic in the wrong layer, a rule duplicated because the
   service layer was bypassed.
8. **Documentation** — every new exported symbol carries the header required by
   `aptlogica-documentation`, and every changed function whose behaviour moved
   has a docstring that still describes it. A stale docstring is worse than none,
   because it is read and believed.

## Severities

Severity is consequence, not effort. "Easy to fix" does not downgrade it; "large
refactor" does not upgrade it.

| Severity | Meaning | Examples |
|---|---|---|
| `critical` | Blocks merge | Missing object-level authz; send path with no approver gate; external content in a system prompt; destructive migration with no rollback; credentials in the diff; a reachable critical CVE |
| `high` | Fix before merge | Unhandled error path on a write; node with no events; unpaginated list; transaction in the wrong layer; agent or auth path with no tests; a new Sonar blocker |
| `medium` | Fix soon; needs owner + follow-up task | N+1 at scale; missing index; test asserting generated prose; logic in a handler; a docstring that contradicts the code |
| `low` | Worth doing | Naming, duplication, unhelpful error message, a missing one-line summary on a new export |
| `info` | No action | Pre-existing issues nearby; a suggested new gate check |

## Verdict

One verdict, with reasoning. A review without a verdict is a comment thread.

- `APPROVE` — no critical, no high.
- `APPROVE_WITH_NITS` — no critical or high, every medium has an owner and a
  follow-up task.
- `REQUEST_CHANGES` — any critical or high.
- `BLOCKED` — gate is red, or the diff is too large to review responsibly.

## Writing the findings

Every critical and high finding needs a file, a line, a rule reference and a
concrete remediation — not "consider improving this".

Every critical finding and every agent-safety finding names the test that should
exist. A security fix with no test gets reverted by the next refactor.

Include positive notes. Not politeness: a review with no positive signal tells
the author nothing about which patterns to repeat, and uniformly hostile reviews
get discounted.

When you find something the gate should have caught, report it *and* propose the
check. A finding you had to make by hand is a check that does not exist yet —
propose it as a diff to `scripts/gate.py` in the review body, for the author to
apply.

## Output shape

```
Verdict: REQUEST_CHANGES

critical  src/api/leads.py:47   Object-level authz missing — `get_lead` filters by
          lead_id only, so any authenticated org can read any lead. Add the
          `org_id` predicate in the repository, not the handler. Test:
          `test_get_lead_cross_org_returns_404`.

high      src/agents/outbound.py:88  Send path has no approver gate...

medium    ...

Positive: the error-path-first ordering in `services/leads.py` is exactly right,
and the eager-load pin on line 62 prevents the N+1 this endpoint would otherwise
have.
```
