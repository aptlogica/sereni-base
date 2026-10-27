# Security scanning — SAST, dependencies, secrets, Sonar

Read this when a scan fails, a CVE lands, a dependency needs upgrading, or you
are wiring scanning into a new repository.

Start with the thing that gets forgotten: **scanners find categories of bug that
are cheap to find automatically. They do not find the bugs most likely to hurt
this product.** No SAST tool will tell you that an endpoint is missing an
object-level authorization check, that a node can reach a tool it should not, or
that a send path has no named-approver gate. Those are review items (SKILL.md
§7) and gate checks we wrote ourselves. Treat scanning as the floor, not the
ceiling — a repository with a green Sonar badge and no authorization tests is
not secure, it is measured.

## The five layers

| Layer | What it catches | Python | Node | Go |
|---|---|---|---|---|
| **SAST** | Injection, unsafe APIs, crypto misuse, code smells | `bandit`, Sonar | `eslint-plugin-security`, Sonar | `gosec` (via golangci-lint), Sonar |
| **SCA** | Known-vulnerable dependencies | `pip-audit` | `pnpm audit`, `osv-scanner` | `govulncheck` |
| **Secrets** | Credentials committed to git | `gitleaks` (language-agnostic, whole history) | | |
| **Containers** | OS-package CVEs in the image | `trivy image` | | |
| **IaC** | Misconfigured Terraform, K8s, Dockerfiles | `trivy config`, `checkov` | | |

`gitleaks` runs over history, not just the diff, because a secret removed in a
later commit is still in the repository and still valid until it is rotated.

`govulncheck` deserves a note: unlike generic SCA it does reachability analysis
and only reports a CVE if your code actually calls the vulnerable symbol. That
cuts the false-positive volume dramatically, and false-positive volume is what
teaches a team to ignore scan output. Where a reachability-aware scanner exists
for a stack, prefer it.

## Severity policy

The gate fails on **critical** and **high** with a known fix available. Medium
and below are recorded and triaged weekly, not merge-blocking — a policy that
blocks on everything gets bypassed within a month, and a bypassed gate is worse
than a permissive one because it is also dishonest.

Two exceptions that block regardless of severity rating:

- **Anything in the auth, approval, or tool-dispatch path.** A medium in the
  code that decides whether a send is authorized is not a medium.
- **Any secret finding.** Always blocking, always followed by rotation. The scan
  finding is not the incident; the credential still being valid is.

## Suppressions

A suppression is a decision, so it is recorded like one. Every suppression
carries an owner, a dated expiry, and a reason:

```toml
# .security-suppressions.toml
[[suppression]]
id      = "GHSA-xxxx-yyyy-zzzz"
scope   = "uv.lock"
reason  = "Vulnerable code path is the XML parser; we never parse untrusted XML."
owner   = "hiraman"
expires = "2026-12-01"
```

The gate fails on an expired suppression. Without an expiry, "we'll deal with it
later" becomes permanent, and the suppressions file becomes the place risk goes
to be forgotten. `reason` explains why it is not exploitable *here* — "not
exploitable" on its own is not a reason, it is an assertion.

## Triage order

When a scan comes back red:

1. **Is it reachable?** Does our code call the vulnerable path? An unreachable
   critical is a suppression with a real reason; a reachable medium may be worse
   than an unreachable critical.
2. **Is there a fixed version?** Upgrade. Most of the time this is the whole
   task and it takes ten minutes.
3. **If no fix exists,** can the vulnerable path be avoided, the dependency
   dropped, or input constrained upstream of it?
4. **If none of the above,** suppress with an owner and an expiry, and file the
   follow-up task. Then say so in the PR — an accepted risk that nobody outside
   the PR knows about is not accepted, it is hidden.

Never bulk-upgrade to clear a scan. A lockfile-wide bump to fix one CVE changes
dozens of transitive dependencies in a diff nobody can review, which trades a
known vulnerability for an unknown supply-chain change.

## Sonar

SonarQube (self-hosted) or SonarCloud, one project per repository, project key
`aptlogica_<repo>`.

Before wiring it in, two honest caveats:

- **It needs infrastructure and a token.** Self-hosted SonarQube is a service to
  run and patch; SonarCloud is a subscription. It is not free in either time or
  money, and for a small team the coverage floors and linters in the per-stack
  references already catch most of what Sonar reports.
- **Sonar's value is trend and consistency across repositories**, not novel
  findings. If we run it, run it everywhere — one repository with a Sonar gate
  and four without gives you a dashboard that lies about the state of the estate.

Configuration:

```properties
# sonar-project.properties
sonar.projectKey=aptlogica_founder-os
sonar.sources=src
sonar.tests=tests
sonar.exclusions=**/migrations/**,**/*_pb2.py,**/node_modules/**
sonar.python.coverage.reportPaths=coverage.xml
sonar.javascript.lcov.reportPaths=coverage/lcov.info
sonar.go.coverage.reportPaths=coverage.out
sonar.qualitygate.wait=true
```

`sonar.qualitygate.wait=true` matters. Without it the scanner uploads and exits
zero regardless of the result, so CI goes green on a failing gate — a scan that
cannot fail the build is telemetry, not a gate.

Quality gate profile: **Sonar way on new code**. Gate on new code, not on the
whole codebase. A total-codebase gate on a decade-old services repository fails
on day one, gets marked "won't fix" wholesale, and stops meaning anything. New
code gates are achievable and therefore get respected.

Reading a Sonar failure:

- **Bug / Vulnerability** on new code — fix it. These are usually real.
- **Security Hotspot** — a human decision, not a defect. Review it, mark it, and
  record the reasoning. Hotspots left unreviewed accumulate and make the gate
  meaningless.
- **Code smell** above the debt threshold — usually a genuinely oversized
  function. Fix it or argue for the threshold, don't suppress the instance.
- **Coverage on new code below threshold** — write the test. This is the most
  common failure and the least worth arguing about.
- **Duplicated blocks** — check whether it is the service-layer bypass from
  SKILL.md §1. A rule implemented twice because someone put logic in a handler
  shows up here as duplication, and the duplication is the symptom.

## Agent-specific supply chain

An agent product has two attack surfaces an ordinary service does not, and no scanner
covers either:

**Skills are executable supply chain.** A SKILL.md is instructions a model
follows and `scripts/` are programs that run with your credentials in a
repository holding client code. Before enabling any third-party skill: read the
SKILL.md and every script, check the licence, and never install a whole domain
bundle. Prefer skills published by the vendor of the thing they cover.

**MCP servers hold live credentials.** Each one is a process with a token,
usually installed with a one-line command. Inventory what is installed, who
installed it, and what scope its token has. An MCP server with a broad OAuth
grant is a bigger exposure than most CVEs you will triage this quarter.

Neither of these shows up in `pnpm audit`. They are quarterly review items with
a named owner.

## Gate integration

`scripts/gate.py` runs SCA and secret scanning per detected stack and, with
`--sonar`, waits on the Sonar quality gate. Scans that need a network or a
running service (container and image scanning, Sonar itself) run in CI rather
than locally — a local gate that takes four minutes stops being run before the
first commit of the day.
