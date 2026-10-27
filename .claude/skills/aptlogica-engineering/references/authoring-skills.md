# Authoring and sourcing skills

Read this when writing a new skill, revising an existing one, or deciding
whether to enable a third-party skill. Universal engineering rules are in
SKILL.md §1–§6.

Skills are versioned artifacts in source control next to the code they govern.
Treat a skill change like a code change.

**Mechanics that are easy to get wrong:**

- The file is `SKILL.md`. Not `SKILLS.md` — that name is invisible to Claude
  Code, and a directory of them silently does nothing.
- Frontmatter carries `name`, `description`, `version`, `allowed-tools`. Bump
  the version on every substantive change.
- `allowed-tools` is a real boundary, so scope it. A review skill gets read-only
  tools; otherwise a reviewer can rewrite the code it was asked to judge.

**Structure — progressive disclosure:**

```
skill-name/
├── SKILL.md          router + invariants, under ~300 lines
├── references/       loaded on demand, one file per stack or variant
└── scripts/          executable gates and deterministic work
```

The SKILL.md body enters context on every trigger. This standard covers three
languages and a security chapter; as one file that would be well over a thousand
lines of mostly irrelevant context per invocation. Routing table in SKILL.md,
concrete patterns in `references/`.

The 300-line budget is a budget, not a target: only two things earn a place
in the always-loaded body — the routing table, and the invariants that change
how code is written no matter which stack you landed in (the agent node
contract, the safety rules, the definition of done). Everything else is
reference. `aptlogica-engineering/SKILL.md` runs about 330 lines because
the agent-safety invariants are load-bearing in all three stacks; that is the
only kind of overrun worth defending, and it is worth re-checking whenever a
section is added.

**Descriptions do the triggering.** All "when to use this" information belongs
in `description`, not the body — the body is only read after selection. Claude
under-triggers skills, so include the casual phrasings a real request uses.

**Enforcement belongs in scripts, not prose.** A checklist in markdown is advice
the model may or may not follow on a long run, and long runs are exactly when it
stops following advice. A `scripts/gate.py` that exits non-zero holds whether or
not anyone read the prose. Explain *why* in prose; enforce *what* in script.

**Write for a model that understands, not one being shouted at.** Piling on
`MUST` and `NEVER` reads as noise and does not survive a long context. State the
rule once and give the consequence.

**Catalog hygiene.** Skill selection degrades as the catalog grows, because the
model picks from descriptions alone. Two skills whose descriptions both read
"reason about the system" means the model guesses. Start specific, consolidate
when overlap appears, and prefer one skill with a routing table over three that
compete.

**Evals.** A skill without a test set cannot be improved deliberately — you
cannot tell whether an edit helped. Keep a handful of realistic prompts per
skill and re-run after a change. This is the weakest part of our setup and the
highest-value thing to add.

**Sourcing third-party skills.** Prefer skills published by the vendor of the
thing they cover (Anthropic, Vercel, HashiCorp, Trail of Bits) over community
mega-catalogs — a named maintainer, a licence, and an incentive to be correct.
Before enabling anything external: read the SKILL.md and every script, check the
licence, and never install a whole domain bundle into a repository holding
client code under NDA. A skill is executable supply chain; treat it like a
dependency, because that is what it is.

