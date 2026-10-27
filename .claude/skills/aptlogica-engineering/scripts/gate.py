#!/usr/bin/env python3
"""Polyglot quality gate for Aptlogica repositories.

Detects which stacks are present (Python, Node/TypeScript, Go), runs the right
toolchain and security scans for each, then applies the cross-language
structural checks from SKILL.md. Exits non-zero when anything fails, so a
standard cannot be quietly ignored.

Exit codes:
    0  all checks passed
    1  at least one check failed
    2  the gate could not run (required tool missing, nothing to check)

Usage:
    python scripts/gate.py
    python scripts/gate.py --report reports/gate.json
    python scripts/gate.py --baseline              # record current state, exit 0
    python scripts/gate.py --stack python          # limit to one stack
    python scripts/gate.py --only lint,types,sca
    python scripts/gate.py --sonar                 # also wait on the Sonar gate
    python scripts/gate.py --allow-missing         # bootstrapping only

Standard library only, so it runs in CI, in a container, and in a bare repo.
"""

from __future__ import annotations

import argparse
import ast
import json
import os
import re
import shutil
import subprocess
import sys
from dataclasses import asdict, dataclass, field
from datetime import date
from pathlib import Path

# --- configuration ----------------------------------------------------------

COVERAGE_FLOOR = 100
AGENT_COVERAGE_FLOOR = 100
BLOCKING_VULN_SEVERITIES = {"critical", "high"}
SUPPRESSIONS_FILE = Path(".security-suppressions.toml")
COVERAGE_EXEMPTIONS_FILE = Path(".coverage-exemptions.toml")
# SereniBase: the root coverage.out is tracked for Sonar; the gate writes its own.
GO_COVERAGE_PROFILE = Path("reports/coverage.out")

# Where a Node/TypeScript project may live. A polyglot repository keeps the
# Python service at the root and the React console under web/, so both stacks
# can use the standard's `src/` layout without colliding.
NODE_ROOTS = (Path("."), Path("web"))

# Directories that hold agent nodes, in any stack.
AGENT_DIRS = ("src/agents", "src/agent", "internal/agent")
# Directories that hold transport handlers, in any stack.
TRANSPORT_DIRS = ("src/api", "src/routers", "src/routes", "internal/transport",
                  "internal/handlers")  # SereniBase

SOURCE_SUFFIXES = {".py", ".ts", ".tsx", ".js", ".mjs", ".go"}
SKIP_DIR_PARTS = {
    "node_modules",
    "__pycache__",
    ".venv",
    "venv",
    "dist",
    "build",
    ".git",
    "vendor",
    "migrations",
    ".next",
    "coverage",
    ".claude",
}
# Command-line tools under scripts/ print to a terminal by design; they run
# no request and no node, so the structured-logging rule does not apply.
CONSOLE_EXEMPT_DIRS = ("scripts",)

# SereniBase: accept a GitHub issue reference (open source) or the internal EPIC/FR form.
TRACEABILITY_RE = re.compile(
    r"Implements:\s*(?:#\d+|EPIC-\d+-S\d+\s*/\s*FR-[A-Z0-9]+-\d+)"
)

HANDLER_RE = {
    ".py": re.compile(r"@(?:router|app)\.(?:get|post|put|patch|delete)\b"),
    ".ts": re.compile(r"\b(?:app|router|fastify)\.(?:get|post|put|patch|delete)\s*\("),
    ".go": re.compile(r"func\s+\([^)]*\)\s+\w+\(w\s+http\.ResponseWriter"),
}
HANDLER_RE[".js"] = HANDLER_RE[".tsx"] = HANDLER_RE[".mjs"] = HANDLER_RE[".ts"]

# Per-language console output that should be structured logging instead.
CONSOLE_RE = {
    ".ts": re.compile(r"(?<![\w.])console\.(log|debug|info|warn|error)\s*\("),
    ".go": re.compile(r"(?<![\w.])fmt\.Print(f|ln)?\s*\("),
}
CONSOLE_RE[".js"] = CONSOLE_RE[".tsx"] = CONSOLE_RE[".mjs"] = CONSOLE_RE[".ts"]

SECRET_ASSIGN_RE = re.compile(
    r"""^\s*(?!.*\#\s*gate:allow-literal)(?!.*//\s*gate:allow-literal)
        (?:const\s+|let\s+|var\s+|)?
        [A-Za-z_.]*(?:password|passwd|secret|token|api_?key|access_?key|private_?key)
        [A-Za-z_]*\s*(?::\s*\w+\s*)?[:=]\s*["'`][^"'`]{6,}["'`]""",
    re.IGNORECASE | re.VERBOSE,
)
CONNECTION_STRING_RE = re.compile(
    r"""["'`](?:postgres|postgresql|mysql|mongodb|redis|amqp)(?:\+\w+)?://[^"'`\s]*:[^"'`@\s]+@""",
    re.IGNORECASE,
)

NODE_EVENT_MARKERS = ("emit_event", "emitEvent", "EmitEvent", "events.emit")
NODE_ALLOWLIST_MARKERS = ("ALLOWED_TOOLS", "AllowedCallers")
NODE_BUDGET_MARKERS = ("TOKEN_BUDGET", "TokenBudget")
NODE_MODEL_MARKERS = ("MODEL", "Model =", "model =")
NODE_CEILING_MARKERS = ("CONTEXT_CEILING", "ContextCeiling")

# Markers indicating a module calls a model, and that it carries provenance.
LLM_CALL_MARKERS = (
    "llm.",
    "llm_client",
    "messages.create",
    "ChatAnthropic",
    "invoke_model",
    "LLM.",
)
PROVENANCE_MARKERS = ("source_id", "sourceId", "SourceID", "Claim", "claims")

PASS, FAIL, MISSING, NA = "PASS", "FAIL", "MISSING", "N/A"


@dataclass
class Check:
    """One check's result: its name, status, one-line detail and any findings."""

    name: str
    status: str
    detail: str = ""
    findings: list[str] = field(default_factory=list)


# --- helpers ----------------------------------------------------------------


def run(cmd: list[str], cwd: Path | None = None) -> tuple[int, str]:
    """Run a command and return its exit code with stdout and stderr merged."""
    proc = subprocess.run(cmd, capture_output=True, text=True, cwd=cwd)
    return proc.returncode, (proc.stdout + proc.stderr).strip()


def node_roots() -> list[Path]:
    """Directories holding a package.json, in NODE_ROOTS order."""
    return [root for root in NODE_ROOTS if (root / "package.json").exists()]


def detect_stacks() -> list[str]:
    """Which stacks live in this repository."""
    stacks = []
    if any(Path(f).exists() for f in ("pyproject.toml", "requirements.txt", "setup.cfg")):
        stacks.append("python")
    if node_roots():
        stacks.append("node")
    if Path("go.mod").exists():
        stacks.append("go")
    return stacks


def source_files(suffixes: set[str] | None = None) -> list[Path]:
    """Every source file we own, excluding vendored and generated trees."""
    wanted = suffixes or SOURCE_SUFFIXES
    out = []
    for p in Path(".").rglob("*"):
        if not p.is_file() or p.suffix not in wanted:
            continue
        if SKIP_DIR_PARTS & set(p.parts):
            continue
        out.append(p)
    return out


def in_any(path: Path, prefixes: tuple[str, ...]) -> bool:
    """Return True when a path sits under one of the given layer directories.

    The prefix is matched as a path segment anywhere in the path, not only at
    the repository root: a pnpm or go workspace keeps the same layers under
    `packages/<name>/src/...`, and an anchored match reported N/A for every
    agent-safety check in such a repo — a gate that goes quiet on a layout is
    worse than one that fails, because it looks green.
    """
    s = "/" + path.as_posix()
    return any(f"/{pre}/" in s or s.endswith(f"/{pre}") for pre in prefixes)


def in_source_tree(path: Path) -> bool:
    """Return True for application source, at the root or inside a workspace package."""
    s = "/" + path.as_posix()
    return any(f"/{root}/" in s for root in ("src", "internal", "app", "lib"))


def tool_check(name: str, cmd: list[str], allow_missing: bool, cwd: Path | None = None) -> Check:
    """Run one external tool as a check, reporting MISSING when it is not installed."""
    if shutil.which(cmd[0]) is None:
        detail = f"{cmd[0]} is not installed"
        if not allow_missing:
            detail += " — install it; the gate cannot vouch for a check it did not run"
        return Check(name, MISSING, detail)
    code, out = run(cmd, cwd=cwd)
    where = f" (in {cwd.as_posix()})" if cwd and cwd != Path(".") else ""
    if code == 0:
        return Check(name, PASS, " ".join(cmd) + where)
    return Check(name, FAIL, " ".join(cmd) + where, out.splitlines()[-40:])


# --- per-stack toolchains ---------------------------------------------------


def python_checks(allow_missing: bool) -> list[Check]:
    """Return the Python toolchain checks: format, lint, types, tests, SCA."""
    return [
        tool_check("py:format", ["ruff", "format", "--check", "."], allow_missing),
        tool_check("py:lint", ["ruff", "check", "."], allow_missing),
        tool_check("py:types", ["mypy", "--strict", "src"], allow_missing),
        tool_check(
            "py:tests",
            [
                "pytest",
                "-q",
                "--cov=src",
                "--cov-report=json",
                "--cov-report=term-missing:skip-covered",
            ],
            allow_missing,
        ),
        tool_check("py:sca", ["pip-audit", "--strict", "--progress-spinner=off"], allow_missing),
    ]


# A repository script that, when declared, is authoritative for that check: the
# package knows its own layout (a pnpm workspace keeps sources in packages/*/src,
# not src/) and the fallback command cannot.
NODE_SCRIPTS = {
    "node:format": ("format", ["prettier", "--check", "src"]),
    "node:lint": ("lint", ["eslint", "src", "--max-warnings", "0"]),
    # `tsc -b` honours a solution-style tsconfig (Vite's default) as well as a flat one.
    "node:types": ("typecheck", ["tsc", "-b", "--noEmit"]),
    "node:tests": ("test", ["vitest", "run", "--coverage"]),
}


def package_scripts(root: Path) -> dict[str, str]:
    """Return the scripts declared in a package.json, empty when unreadable."""
    try:
        return json.loads((root / "package.json").read_text(encoding="utf-8")).get("scripts", {})
    except (OSError, ValueError):
        return {}


def node_checks(allow_missing: bool) -> list[Check]:
    """Return the Node toolchain checks, preferring the package's own scripts."""
    checks: list[Check] = []
    for root in node_roots():
        pm = "pnpm" if (root / "pnpm-lock.yaml").exists() else "npm"
        exec_prefix = [pm, "exec"] if pm == "pnpm" else ["npx", "--no-install"]
        scripts = package_scripts(root)
        for name, (script, fallback) in NODE_SCRIPTS.items():
            command = [pm, "run", script] if script in scripts else [*exec_prefix, *fallback]
            checks.append(tool_check(name, command, allow_missing, cwd=root))
        checks.append(
            tool_check("node:sca", [pm, "audit", "--audit-level=high"], allow_missing, cwd=root)
        )
    return checks


def go_checks(allow_missing: bool) -> list[Check]:
    """Return the Go toolchain checks: fmt, vet, lint, race tests, vulncheck."""
    # SereniBase: a profile left by an earlier run must not be read as this run's.
    GO_COVERAGE_PROFILE.parent.mkdir(parents=True, exist_ok=True)
    GO_COVERAGE_PROFILE.unlink(missing_ok=True)
    checks = [
        tool_check("go:vet", ["go", "vet", "./..."], allow_missing),
        tool_check("go:lint", ["golangci-lint", "run"], allow_missing),
        # SereniBase: tests live in tests/, so measure coverage of internal/ the way CI does.
        tool_check("go:tests", ["go", "test", "-race", f"-coverprofile={GO_COVERAGE_PROFILE.as_posix()}",
                                "-covermode=atomic", "-coverpkg=./internal/...",
                                "./tests/..."], allow_missing),
        tool_check("go:sca", ["govulncheck", "./..."], allow_missing),
    ]
    # gofmt reports by printing filenames rather than a non-zero exit.
    if shutil.which("gofmt") is None:
        checks.insert(0, Check("go:format", MISSING, "gofmt is not installed"))
    else:
        _code, out = run(["gofmt", "-l", "."])
        unformatted = [
            line
            for line in out.splitlines()
            if line.strip() and not SKIP_DIR_PARTS & set(Path(line).parts)
        ]
        checks.insert(
            0,
            Check(
                "go:format",
                FAIL if unformatted else PASS,
                "gofmt -l .",
                unformatted,
            ),
        )
    return checks


STACK_CHECKS = {"python": python_checks, "node": node_checks, "go": go_checks}


# --- cross-language structural checks ---------------------------------------


def check_no_console() -> Check:
    """No print/console.log/fmt.Print in application code (SKILL.md §1)."""
    findings = []
    for path in source_files():
        if in_any(path, CONSOLE_EXEMPT_DIRS):
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        if path.suffix == ".py":
            try:
                tree = ast.parse(text)
            except SyntaxError as exc:
                findings.append(f"{path}: unparseable ({exc.msg} line {exc.lineno})")
                continue
            for node in ast.walk(tree):
                if (
                    isinstance(node, ast.Call)
                    and isinstance(node.func, ast.Name)
                    and node.func.id == "print"
                ):
                    findings.append(f"{path}:{node.lineno}: print() — use structlog")
            continue
        pattern = CONSOLE_RE.get(path.suffix)
        if pattern is None:
            continue
        for lineno, line in enumerate(text.splitlines(), start=1):
            if "gate:allow-console" in line:
                continue
            if pattern.search(line):
                findings.append(f"{path}:{lineno}: unstructured console output")
    return Check(
        "no-console", FAIL if findings else PASS, "structured logging only (SKILL.md §1)", findings
    )


def check_python_exceptions() -> Check:
    """No bare or swallowed exceptions. Python-specific by nature."""
    files = [p for p in source_files({".py"})]
    if not files:
        return Check("py:exceptions", NA, "no Python sources")
    findings = []
    for path in files:
        try:
            tree = ast.parse(path.read_text(encoding="utf-8", errors="replace"))
        except SyntaxError:
            continue
        for node in ast.walk(tree):
            if not isinstance(node, ast.ExceptHandler):
                continue
            if node.type is None:
                findings.append(f"{path}:{node.lineno}: bare except")
                continue
            broad = isinstance(node.type, ast.Name) and node.type.id in {
                "Exception",
                "BaseException",
            }
            swallows = all(
                isinstance(s, ast.Pass)
                or (isinstance(s, ast.Expr) and isinstance(s.value, ast.Constant))
                for s in node.body
            )
            if broad and swallows:
                findings.append(f"{path}:{node.lineno}: except {node.type.id} with no handling")
    return Check(
        "py:exceptions",
        FAIL if findings else PASS,
        "errors handled or re-raised (SKILL.md §1)",
        findings,
    )


def check_go_discarded_errors() -> Check:
    """`_ = f()` without a justifying comment. Go-specific by nature."""
    files = source_files({".go"})
    if not files:
        return Check("go:discarded-errors", NA, "no Go sources")
    findings = []
    pattern = re.compile(r"^\s*_\s*=\s*\w[\w.]*\(")
    for path in files:
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
        for i, line in enumerate(lines):
            if not pattern.match(line):
                continue
            context = " ".join(lines[max(0, i - 1) : i + 1])
            if "//" not in context:
                findings.append(f"{path}:{i + 1}: discarded error with no explanatory comment")
    return Check(
        "go:discarded-errors",
        FAIL if findings else PASS,
        "a discarded error is indistinguishable from success",
        findings,
    )


def check_no_hardcoded_secrets() -> Check:
    """Fail on a credential literal or a credentialed connection string in source."""
    findings = []
    for path in source_files():
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
        for lineno, line in enumerate(lines, start=1):
            previous = lines[lineno - 2] if lineno > 1 else ""
            if "gate:allow-literal" in line or "gate:allow-literal" in previous:
                continue
            if SECRET_ASSIGN_RE.match(line):
                findings.append(f"{path}:{lineno}: literal assigned to a secret name")
            elif CONNECTION_STRING_RE.search(line):
                findings.append(f"{path}:{lineno}: connection string with credentials")
    return Check(
        "no-hardcoded-secrets",
        FAIL if findings else PASS,
        "configuration is loaded, not literal (SKILL.md §1)",
        findings,
    )


def check_traceability() -> Check:
    """Every transport handler and every node cites a requirement (SKILL.md §6)."""
    targets = [p for p in source_files() if in_any(p, TRANSPORT_DIRS) or in_any(p, AGENT_DIRS)]
    if not targets:
        return Check(
            "traceability",
            NA,
            f"no transport or agent directory found ({', '.join(TRANSPORT_DIRS + AGENT_DIRS)}, "
            "at the root or under packages/*)",
        )
    findings = []
    for path in targets:
        text = path.read_text(encoding="utf-8", errors="replace")
        lines = text.splitlines()
        pattern = HANDLER_RE.get(path.suffix)
        hits = [i for i, ln in enumerate(lines) if pattern and pattern.search(ln)]
        if hits:
            for i in hits:
                window = "\n".join(lines[max(0, i - 4) : i + 8])
                if not TRACEABILITY_RE.search(window):
                    findings.append(
                        f"{path}:{i + 1}: handler without 'Implements: EPIC-XX-SXX / FR-MODULE-NNN'"
                    )
        elif in_any(path, AGENT_DIRS) and not path.name.startswith(("__init__", "index")):
            if not TRACEABILITY_RE.search(text):
                findings.append(f"{path}: node module without an Implements comment")
    return Check(
        "traceability",
        FAIL if findings else PASS,
        "requirements provable, not asserted (SKILL.md §6)",
        findings,
    )


def check_node_contracts() -> Check:
    """Nodes declare an allowlist and a budget, and emit events (SKILL.md §3)."""
    modules = [
        p
        for p in source_files()
        if in_any(p, AGENT_DIRS)
        and not p.name.startswith(("__init__", "index"))
        and not re.search(r"(_test|\.test|\.spec)\.", p.name)
    ]
    if not modules:
        return Check("node-contracts", NA, "no agent directory present")
    findings = []
    for path in modules:
        text = path.read_text(encoding="utf-8", errors="replace")
        if not any(m in text for m in NODE_EVENT_MARKERS):
            findings.append(f"{path}: emits no lifecycle event — invisible to the UI and audit log")
        if not any(m in text for m in NODE_ALLOWLIST_MARKERS):
            findings.append(f"{path}: no tool allowlist declared")
        if not any(m in text for m in NODE_BUDGET_MARKERS):
            findings.append(f"{path}: no token budget declared")
    return Check(
        "node-contracts", FAIL if findings else PASS, "agent node contracts (SKILL.md §3)", findings
    )


def _agent_modules() -> list[Path]:
    return [
        p
        for p in source_files()
        if in_any(p, AGENT_DIRS)
        and not p.name.startswith(("__init__", "index"))
        and not re.search(r"(_test|\.test|\.spec)\.", p.name)
    ]


def check_node_limits() -> Check:
    """Every node declares a model and a context ceiling (SKILL.md §5).

    TOKEN_BUDGET caps spend; CONTEXT_CEILING caps input length. They are
    different failures and the gate wants both declared explicitly, because an
    undeclared ceiling means whatever the retriever happened to return.

    Applies only to modules that actually call a model — a tool server or
    worker that never does is not asked to declare either.
    """
    modules = _agent_modules()
    if not modules:
        return Check("node-limits", NA, "no agent directory present")
    findings = []
    for path in modules:
        text = path.read_text(encoding="utf-8", errors="replace")
        if not any(m in text for m in LLM_CALL_MARKERS):
            continue  # a tool server with no model call needs neither
        if not any(m in text for m in NODE_MODEL_MARKERS):
            findings.append(
                f"{path}: no MODEL declared — cheap models should be routed "
                "deliberately, not by default"
            )
        if not any(m in text for m in NODE_CEILING_MARKERS):
            findings.append(f"{path}: no CONTEXT_CEILING declared")
    return Check(
        "node-limits",
        FAIL if findings else PASS,
        "model routing and context ceilings (SKILL.md §5)",
        findings,
    )


def check_provenance() -> Check:
    """A module that calls a model carries provenance on its output (SKILL.md §5).

    This checks that a claims structure is referenced, not that every sentence
    maps to a source — that needs a verifier pass, not a regex. It catches the
    node that returns bare prose, which is the case that matters.
    """
    modules = _agent_modules()
    if not modules:
        return Check("provenance", NA, "no agent directory present")
    findings = []
    for path in modules:
        text = path.read_text(encoding="utf-8", errors="replace")
        if not any(m in text for m in LLM_CALL_MARKERS):
            continue  # no model call, nothing to ground
        if not any(m in text for m in PROVENANCE_MARKERS):
            findings.append(
                f"{path}: calls a model but references no source_id or claims "
                "structure — output would be unattributable"
            )
    return Check(
        "provenance",
        FAIL if findings else PASS,
        "generated fact carries its source (SKILL.md §5)",
        findings,
    )


def check_migration_present() -> Check:
    """A schema change and its migration land in the same commit (SKILL.md §2)."""
    if shutil.which("git") is None or not Path(".git").exists():
        return Check("migration-present", NA, "not a git repository")
    code, out = run(["git", "diff", "--cached", "--name-only", "HEAD"])
    if code != 0 or not out:
        code, out = run(["git", "diff", "--name-only", "HEAD"])
    if code != 0:
        return Check("migration-present", NA, "no HEAD to diff against")
    changed = [line for line in out.splitlines() if line.strip()]
    # A brand-new model or migration file is untracked and absent from `git diff`;
    # without this a new migration looks missing and a new model goes unchecked.
    code, untracked = run(["git", "ls-files", "--others", "--exclude-standard"])
    if code == 0:
        changed += [line for line in untracked.splitlines() if line.strip()]
    if not changed:
        return Check("migration-present", NA, "no changes to inspect")

    schema_changed = [
        f
        for f in changed
        if (
            re.search(r"(models?|schema|entities)[^/]*\.(py|ts|go)$", f)
            or "/models/" in f
            or f.endswith("schema.prisma")
        )
        # Generated OpenAPI types (web/src/api/schema.d.ts) describe the HTTP contract, not a
        # stored schema; the openapi-types check owns them.
        and not f.endswith(".d.ts")
    ]
    migration_added = [
        f
        for f in changed
        if "alembic/versions" in f or "/migrations/" in f or "prisma/migrations" in f
    ]
    if schema_changed and not migration_added:
        return Check(
            "migration-present",
            FAIL,
            "schema changed without a migration (SKILL.md §2)",
            schema_changed,
        )
    return Check("migration-present", PASS, f"{len(migration_added)} migration file(s)")


def check_suppressions() -> Check:
    """No expired security suppressions (security-scanning.md)."""
    if not SUPPRESSIONS_FILE.exists():
        return Check("suppressions", NA, f"no {SUPPRESSIONS_FILE}")
    text = SUPPRESSIONS_FILE.read_text(encoding="utf-8")
    blocks = re.split(r"\[\[suppression\]\]", text)[1:]
    findings, today = [], date.today()
    for block in blocks:

        def field_of(key: str) -> str:
            """Return one quoted field from this suppression block, or an empty string."""
            m = re.search(rf'^\s*{key}\s*=\s*"([^"]*)"', block, re.M)  # noqa: B023 — same iteration
            return m.group(1) if m else ""

        sid = field_of("id") or "<no id>"
        for key in ("owner", "reason", "expires"):
            if not field_of(key):
                findings.append(f"{sid}: missing {key}")
        expires = field_of("expires")
        if expires:
            try:
                if date.fromisoformat(expires) < today:
                    findings.append(f"{sid}: expired {expires} — re-triage or renew")
            except ValueError:
                findings.append(f"{sid}: expires is not an ISO date ({expires!r})")
    return Check(
        "suppressions", FAIL if findings else PASS, f"{len(blocks)} suppression(s)", findings
    )


def check_secret_scan(allow_missing: bool) -> Check:
    """gitleaks over history, not just the working tree."""
    if shutil.which("gitleaks") is None:
        detail = "gitleaks is not installed"
        if not allow_missing:
            detail += " — a committed secret stays valid until it is rotated"
        return Check("secret-scan", MISSING, detail)
    if not Path(".git").exists():
        return Check("secret-scan", NA, "not a git repository")
    code, out = run(["gitleaks", "detect", "--no-banner", "--redact"])
    if code == 0:
        return Check("secret-scan", PASS, "gitleaks detect (full history)")
    return Check(
        "secret-scan", FAIL, "gitleaks found secrets — rotate, then remove", out.splitlines()[-30:]
    )


def _coverage_from_pytest(data: dict) -> tuple[float, dict[str, float]]:
    """Return (total, per-file) from a coverage.py JSON report."""
    per_file = {
        name: f.get("summary", {}).get("percent_covered", 0.0)
        for name, f in data.get("files", {}).items()
    }
    return data.get("totals", {}).get("percent_covered", 0.0), per_file


def _coverage_from_istanbul(data: dict) -> tuple[float, dict[str, float]]:
    """Return (total, per-file) from an Istanbul/v8 coverage-final.json.

    Line counters are keyed by statement id, so the percentage is computed the
    same way the JS reporters compute it rather than read off a field.
    """
    per_file: dict[str, float] = {}
    covered_total = found_total = 0
    for name, entry in data.items():
        counters = entry.get("s", {})
        if not isinstance(counters, dict):
            continue
        found = len(counters)
        covered = sum(1 for hits in counters.values() if hits)
        found_total += found
        covered_total += covered
        rel = os.path.relpath(name, os.getcwd()) if os.path.isabs(name) else name
        per_file[rel] = 100.0 * covered / found if found else 100.0
    total = 100.0 * covered_total / found_total if found_total else 0.0
    return total, per_file


def _coverage_from_go(text: str) -> tuple[float, dict[str, float]]:
    """Return (total, per-file) from a `go test -coverprofile` file.

    With `-coverpkg`, each test binary writes its own copy of every block, so a
    block counts as covered when any copy of it ran. Paths are made relative to
    the module root so exemption paths match them.
    """
    module = ""
    if Path("go.mod").exists():
        m = re.search(r"^module\s+(\S+)", Path("go.mod").read_text(encoding="utf-8"), re.M)
        module = m.group(1) + "/" if m else ""
    blocks: dict[str, tuple[int, bool]] = {}
    for line in text.splitlines():
        parts = line.rsplit(" ", 2)
        if line.startswith("mode:") or len(parts) != 3:
            continue
        location, statements, count = parts
        seen = blocks.get(location, (int(statements), False))
        blocks[location] = (seen[0], seen[1] or int(count) > 0)
    found: dict[str, int] = {}
    covered: dict[str, int] = {}
    for location, (statements, hit) in blocks.items():
        name = location.split(":")[0].removeprefix(module)
        found[name] = found.get(name, 0) + statements
        covered[name] = covered.get(name, 0) + (statements if hit else 0)
    per_file = {n: 100.0 * covered[n] / found[n] if found[n] else 100.0 for n in found}
    total_found = sum(found.values())
    total = 100.0 * sum(covered.values()) / total_found if total_found else 0.0
    return total, per_file


def coverage_exemptions() -> tuple[list[tuple[str, float]], list[str]]:
    """Return (path, floor) pairs from `.coverage-exemptions.toml`, plus findings.

    A coverage floor with no legitimate way down gets argued away under deadline
    pressure, and an argued-away floor takes the whole gate's credibility with
    it. An exemption is the legitimate way: it names a path, a lower floor, an
    owner, a reason and a dated expiry, and the gate fails once it expires —
    the same shape as a security suppression, for the same reason.

        [[exemption]]
        path    = "packages/runner/src/adapters"
        floor   = 85
        reason  = "Provider SDK abort paths need a live socket to exercise."
        owner   = "aniket"
        expires = "2026-12-01"
    """
    if not COVERAGE_EXEMPTIONS_FILE.exists():
        return [], []
    text = COVERAGE_EXEMPTIONS_FILE.read_text(encoding="utf-8")
    exemptions: list[tuple[str, float]] = []
    findings: list[str] = []
    today = date.today()
    for block in re.split(r"\[\[exemption\]\]", text)[1:]:

        def value_of(key: str) -> str:
            """Return one field from this exemption block, or an empty string."""
            m = re.search(rf'^\s*{key}\s*=\s*"?([^"\n]*)"?', block, re.M)  # noqa: B023 — same iteration
            return m.group(1).strip() if m else ""

        path = value_of("path") or "<no path>"
        for key in ("owner", "reason", "expires", "floor"):
            if not value_of(key):
                findings.append(f"coverage exemption {path}: missing {key}")
        expires = value_of("expires")
        if expires:
            try:
                if date.fromisoformat(expires) < today:
                    findings.append(f"coverage exemption {path} expired {expires}")
                    continue
            except ValueError:
                findings.append(f"coverage exemption {path}: unreadable expiry {expires!r}")
                continue
        try:
            floor = float(value_of("floor"))
        except ValueError:
            continue
        if any(d in path for d in AGENT_DIRS):
            findings.append(
                f"coverage exemption {path} covers agent code — allowed, but say so in review"
            )
        exemptions.append((path, floor))
    return exemptions, findings


def check_coverage() -> Check:
    """Read whichever coverage report the stack produced.

    Python writes `coverage.json`; vitest and jest write `coverage/`. A gate that
    reports N/A because it only knew one of those shapes is a floor nobody is
    standing on, which is worse than having no floor.
    """
    sources = (
        (Path("coverage.json"), _coverage_from_pytest),
        (Path("coverage/coverage-final.json"), _coverage_from_istanbul),
        (Path("web/coverage/coverage-final.json"), _coverage_from_istanbul),
        (GO_COVERAGE_PROFILE, _coverage_from_go),  # SereniBase
    )
    reports = [(path, reader) for path, reader in sources if path.exists()]
    if not reports:
        return Check(
            "coverage", NA, "no coverage report — run tests with a JSON coverage reporter"
        )

    exemptions, findings = coverage_exemptions()
    totals: list[str] = []

    def floor_for(name: str) -> float:
        """Return the floor that applies to one file: an exemption, or the default."""
        for prefix, floor in exemptions:
            if prefix in name:
                return floor
        return AGENT_COVERAGE_FLOOR if any(d in name for d in AGENT_DIRS) else COVERAGE_FLOOR

    for path, reader in reports:
        try:
            text = path.read_text(encoding="utf-8")
            # SereniBase: a Go profile is plain text; the other reports are JSON.
            total, per_file = reader(text if path.suffix == ".out" else json.loads(text))
        except (OSError, ValueError) as exc:
            findings.append(f"{path} could not be read: {exc}")
            continue
        totals.append(f"{path}: {total:.1f}%")
        exempt = [n for n in per_file if any(prefix in n for prefix, _ in exemptions)]
        # The repository-wide total is only meaningful when nothing is exempt;
        # with exemptions the per-file floors are the check, and the total is
        # reported rather than enforced so an exempt path cannot drag it red.
        if not exempt and total < COVERAGE_FLOOR:
            findings.append(f"{path} total {total:.1f}% < {COVERAGE_FLOOR}%")
        for name, pct in sorted(per_file.items()):
            floor = floor_for(name)
            if pct < floor:
                findings.append(f"{name} {pct:.1f}% < {floor:.0f}%")
    return Check(
        "coverage",
        FAIL if findings else PASS,
        f"{'; '.join(totals)} (floor {COVERAGE_FLOOR}%, agents {AGENT_COVERAGE_FLOOR}%"
        + (f", {len(exemptions)} exempt paths)" if exemptions else ")"),
        findings,
    )


def check_sonar(allow_missing: bool) -> Check:
    """Run the Sonar scanner and wait on the quality gate."""
    if not Path("sonar-project.properties").exists():
        return Check("sonar", NA, "no sonar-project.properties")
    if shutil.which("sonar-scanner") is None:
        return Check("sonar", MISSING, "sonar-scanner is not installed")
    code, out = run(["sonar-scanner", "-Dsonar.qualitygate.wait=true"])
    if code == 0:
        return Check("sonar", PASS, "quality gate passed")
    return Check("sonar", FAIL, "quality gate failed", out.splitlines()[-30:])


# Scope §7 / D-042: only these may build a CompletionRequest or reach a provider.
# The only modules that may reach a provider: the gateway, the context assembler that builds every
# prompt, and the single node-facing seam that assembles, routes, calls and records a step (D-048).
PROMPT_OWNERS = ("src/orchestration/context.py", "src/gateway", "src/services/inference.py")
PROMPT_FORBIDDEN_MODULES = ("src.gateway.gateway", "src.gateway.factory", "anthropic", "openai")
PROMPT_FORBIDDEN_NAMES = {"CompletionRequest", "Message"}


def check_prompt_boundary() -> Check:
    """No node builds a prompt or calls a provider; the context assembler does (scope §7)."""
    findings = []
    for path in source_files({".py"}):
        posix = path.as_posix()
        if not in_source_tree(path) or in_any(path, PROMPT_OWNERS):
            continue
        try:
            tree = ast.parse(path.read_text(encoding="utf-8", errors="replace"))
        except SyntaxError as exc:
            findings.append(f"{path}: unparseable ({exc.msg} line {exc.lineno})")
            continue
        is_node = in_any(path, AGENT_DIRS)
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):
                modules = [alias.name for alias in node.names]
            elif isinstance(node, ast.ImportFrom):
                modules = [node.module or ""]
                if node.module == "src.gateway.types":
                    for alias in node.names:
                        if alias.name in PROMPT_FORBIDDEN_NAMES:
                            findings.append(
                                f"{path}:{node.lineno}: imports {alias.name} — only the context "
                                "assembler builds prompts (scope §7)"
                            )
            else:
                if (
                    is_node
                    and isinstance(node, ast.Call)
                    and isinstance(node.func, ast.Attribute)
                    and node.func.attr == "complete"
                ):
                    findings.append(
                        f"{path}:{node.lineno}: calls .complete() — nodes go through the assembler"
                    )
                continue
            for module in modules:
                if any(module == m or module.startswith(m + ".") for m in PROMPT_FORBIDDEN_MODULES):
                    findings.append(
                        f"{path}:{node.lineno}: imports {module} — only the gateway and the "
                        "context assembler may reach a provider (scope §7)"
                    )
    return Check(
        "prompt-boundary",
        FAIL if findings else PASS,
        "no node builds a prompt or calls a provider (scope §7)",
        findings,
    )


# D-005 / D-045: checkpoint replay depends on node purity. A node module may hold imports,
# functions, classes and UPPER_CASE constants at module level — nothing instantiated, nothing
# mutable — and may never write to the `state` it receives.
PURE_MODULE_CONSTANT_RE = re.compile(r"^[A-Z][A-Z0-9_]*$")
STATE_MUTATORS = {"update", "pop", "setdefault", "clear", "append", "extend", "insert", "remove"}
NODE_TEST_DIR = Path("tests/agents")


def _is_constant_value(node: ast.AST) -> bool:
    if isinstance(node, ast.Constant):
        return True
    if isinstance(node, (ast.Tuple, ast.List, ast.Set)):
        return all(_is_constant_value(e) for e in node.elts)
    if isinstance(node, ast.Dict):
        return all(k is not None and _is_constant_value(k) for k in node.keys) and all(
            _is_constant_value(v) for v in node.values
        )
    return False


def _state_names(func: ast.FunctionDef | ast.AsyncFunctionDef) -> set[str]:
    names = {a.arg for a in func.args.args[:1]}  # the first parameter is the state
    names.add("state")
    return names


def _touches_state(target: ast.AST, names: set[str]) -> bool:
    node = target
    while isinstance(node, (ast.Subscript, ast.Attribute)):
        node = node.value
    return isinstance(node, ast.Name) and node.id in names and node is not target


def check_node_purity() -> Check:
    """Nodes are pure: no module-level mutable state, no mutation of `state` (SKILL.md §3)."""
    modules = _agent_modules()
    if not modules:
        return Check("node-purity", NA, "no agent directory present")
    findings = []
    for path in modules:
        try:
            tree = ast.parse(path.read_text(encoding="utf-8", errors="replace"))
        except SyntaxError as exc:
            findings.append(f"{path}: unparseable ({exc.msg} line {exc.lineno})")
            continue
        for stmt in tree.body:
            if isinstance(
                stmt,
                (ast.Import, ast.ImportFrom, ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef),
            ):
                continue
            if isinstance(stmt, ast.Expr) and isinstance(stmt.value, ast.Constant):
                continue  # docstring
            if isinstance(stmt, ast.If):
                continue  # TYPE_CHECKING guards
            if isinstance(stmt, (ast.Assign, ast.AnnAssign)):
                targets = stmt.targets if isinstance(stmt, ast.Assign) else [stmt.target]
                names = [t.id for t in targets if isinstance(t, ast.Name)]
                value = stmt.value
                if (
                    len(names) == len(targets)
                    and all(PURE_MODULE_CONSTANT_RE.match(n) for n in names)
                    and value is not None
                    and _is_constant_value(value)
                ):
                    continue
                findings.append(
                    f"{path}:{stmt.lineno}: module-level state — only UPPER_CASE constants of "
                    "literals are allowed; a client or mutable value here breaks checkpoint replay"
                )
                continue
            findings.append(
                f"{path}:{stmt.lineno}: module-level statement is not allowed in a node"
            )
        for node in ast.walk(tree):
            if isinstance(node, (ast.Global, ast.Nonlocal)):
                findings.append(f"{path}:{node.lineno}: global/nonlocal in a node")
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                names = _state_names(node)
                for inner in ast.walk(node):
                    targets: list[ast.AST] = []
                    if isinstance(inner, (ast.Assign, ast.AugAssign, ast.AnnAssign)):
                        targets = inner.targets if isinstance(inner, ast.Assign) else [inner.target]
                    if isinstance(inner, ast.Delete):
                        targets = list(inner.targets)
                    for target in targets:
                        if _touches_state(target, names):
                            findings.append(
                                f"{path}:{inner.lineno}: writes to state — return a delta instead"
                            )
                    if (
                        isinstance(inner, ast.Call)
                        and isinstance(inner.func, ast.Attribute)
                        and inner.func.attr in STATE_MUTATORS
                        and _touches_state(inner.func, names)
                    ):
                        findings.append(
                            f"{path}:{inner.lineno}: mutates state via .{inner.func.attr}()"
                        )
        test_file = NODE_TEST_DIR / f"test_{path.stem}.py"
        if not test_file.exists():
            findings.append(f"{path}: no {test_file} — every node ships a replay and sequence test")
        else:
            text = test_file.read_text(encoding="utf-8", errors="replace")
            for marker in ("replay", "sequence"):
                if not re.search(rf"def test_\w*{marker}\w*", text):
                    findings.append(f"{test_file}: no test named *{marker}* for {path.stem}")
    return Check(
        "node-purity",
        FAIL if findings else PASS,
        "pure nodes with a replay and an event-sequence test (D-005)",
        findings,
    )


def check_openapi_types() -> Check:
    """The console's generated API types match the application's OpenAPI document (D-046)."""
    if not Path("web/package.json").exists() or not Path("scripts/openapi.py").exists():
        return Check("openapi-types", NA, "no console or no generator present")
    if shutil.which("pnpm") is None:
        return Check("openapi-types", MISSING, "pnpm is not installed")
    code, out = run([sys.executable, "scripts/openapi.py", "--check"])
    return Check(
        "openapi-types",
        PASS if code == 0 else FAIL,
        "web/src/api/schema.d.ts regenerated from the app and diffed (D-046)",
        [] if code == 0 else out.splitlines()[-10:],
    )


STRUCTURAL = {
    "no-console": check_no_console,
    "openapi-types": check_openapi_types,
    "prompt-boundary": check_prompt_boundary,
    "node-purity": check_node_purity,
    "py:exceptions": check_python_exceptions,
    "go:discarded-errors": check_go_discarded_errors,
    "no-hardcoded-secrets": check_no_hardcoded_secrets,
    "traceability": check_traceability,
    "node-contracts": check_node_contracts,
    "node-limits": check_node_limits,
    "provenance": check_provenance,
    "migration-present": check_migration_present,
    "suppressions": check_suppressions,
    "coverage": check_coverage,
}


# --- orchestration ----------------------------------------------------------


def main() -> int:
    """Run the gate and return its exit code: 0 pass, 1 fail, 2 could not run."""
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--report", metavar="PATH", help="write JSON results here")
    ap.add_argument(
        "--baseline", action="store_true", help="record current state and always exit 0"
    )
    ap.add_argument("--stack", help="limit to one stack: python, node or go")
    ap.add_argument("--only", help="comma-separated check names")
    ap.add_argument("--sonar", action="store_true", help="also wait on the Sonar gate")
    ap.add_argument(
        "--allow-missing",
        action="store_true",
        help="treat an uninstalled tool as skipped (bootstrapping only)",
    )
    args = ap.parse_args()

    stacks = detect_stacks()
    if args.stack:
        if args.stack not in STACK_CHECKS:
            print(f"gate: unknown stack {args.stack!r}", file=sys.stderr)
            return 2
        stacks = [args.stack] if args.stack in stacks else []
        if not stacks:
            print(f"gate: no {args.stack} manifest in this repository", file=sys.stderr)
            return 2
    if not stacks:
        print("gate: cannot run — no pyproject.toml, package.json or go.mod found", file=sys.stderr)
        print("gate: run this from the repository root", file=sys.stderr)
        return 2

    checks: list[Check] = []
    for stack in stacks:
        checks += STACK_CHECKS[stack](args.allow_missing)
    checks += [fn() for fn in STRUCTURAL.values()]
    checks.append(check_secret_scan(args.allow_missing))
    if args.sonar:
        checks.append(check_sonar(args.allow_missing))

    if args.only:
        wanted = {n.strip() for n in args.only.split(",")}
        checks = [c for c in checks if c.name in wanted]
    if not checks:
        print("gate: no checks selected", file=sys.stderr)
        return 2

    print(f"gate: stacks detected — {', '.join(stacks)}\n")
    width = max(len(c.name) for c in checks)
    for c in checks:
        print(f"  {c.status:<7} {c.name:<{width}}  {c.detail}")
        for finding in c.findings[:12]:
            print(f"          - {finding}")
        if len(c.findings) > 12:
            print(f"          ... and {len(c.findings) - 12} more")

    failed = [c for c in checks if c.status == FAIL]
    missing = [c for c in checks if c.status == MISSING]
    verdict = "PASS" if not failed and (args.allow_missing or not missing) else "FAIL"

    payload = {
        "verdict": verdict,
        "stacks": stacks,
        "checks": [asdict(c) for c in checks],
        "failed": [c.name for c in failed],
        "missing": [c.name for c in missing],
    }
    report_path = args.report or ("reports/gate-baseline.json" if args.baseline else None)
    if report_path:
        Path(report_path).parent.mkdir(parents=True, exist_ok=True)
        Path(report_path).write_text(json.dumps(payload, indent=2) + "\n")
        print(f"\ngate: wrote {report_path}")

    if args.baseline:
        print(f"gate: baseline recorded ({verdict}) — pre-existing failures are not yours")
        return 0
    if missing and not args.allow_missing:
        print(
            f"\ngate: CANNOT RUN — missing tools: {', '.join(c.name for c in missing)}",
            file=sys.stderr,
        )
        return 2
    if failed:
        print(f"\ngate: FAIL — {', '.join(c.name for c in failed)}", file=sys.stderr)
        return 1
    print("\ngate: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
