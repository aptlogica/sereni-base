#!/usr/bin/env python3
"""Documentation gate for Aptlogica repositories.

Finds what the documentation standard requires and the code does not have: a
file header, a summary on every exported class and function, a `Called from:`
line on exports in the layered directories, and the tool/budget/emits block on
agent nodes. Also regenerates `Called from:` lists, so nobody has to guess them.

It checks presence, never truth. Whether the sentence is correct is a review
item (`aptlogica-review`, item 8).

Exit codes:
    0  nothing missing
    1  findings
    2  could not run (no source files, unreadable path)

Usage:
    python scripts/docgate.py
    python scripts/docgate.py --stack python
    python scripts/docgate.py --paths src/agents src/services
    python scripts/docgate.py --report reports/docs.json
    python scripts/docgate.py --callers src/services/leads.py

Standard library only, so it runs in CI, in a container, and in a bare repo.
"""

from __future__ import annotations

import argparse
import ast
import json
import re
import sys
from dataclasses import asdict, dataclass
from pathlib import Path

# --- configuration ----------------------------------------------------------

PY_SUFFIXES = {".py"}
TS_SUFFIXES = {".ts", ".tsx", ".js", ".jsx", ".mjs"}
GO_SUFFIXES = {".go"}
SOURCE_SUFFIXES = PY_SUFFIXES | TS_SUFFIXES | GO_SUFFIXES

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
    "graphify-out",
}

# Directories where an export is called from somewhere worth naming. Outside
# these, a missing `Called from:` line is not worth failing a build over.
LAYERED_DIRS = (
    "src/api",
    "src/routers",
    "src/routes",
    "src/services",
    "src/repositories",
    "src/agents",
    "src/agent",
    "internal/transport",
    "internal/service",
    "internal/repository",
    "internal/agent",
    "internal/handlers",  # SereniBase
    "internal/services",  # SereniBase
)
AGENT_DIRS = ("src/agents", "src/agent", "internal/agent")

CALLED_FROM_RE = re.compile(r"Called from:", re.IGNORECASE)
NODE_BLOCK_RE = re.compile(r"Tools:.*\n.*Budget:", re.IGNORECASE | re.DOTALL)

TS_EXPORT_RE = re.compile(
    r"^export\s+(?:default\s+)?"
    r"(?:async\s+)?(?P<kind>function|class|interface|type|const|enum)\s+"
    r"(?P<name>[A-Za-z_$][\w$]*)"
)
GO_FUNC_RE = re.compile(r"^func\s+(?:\([^)]*\)\s*)?(?P<name>[A-Z]\w*)\s*[\(\[]")
GO_TYPE_RE = re.compile(r"^(?:type|var|const)\s+(?P<name>[A-Z]\w*)\b")

# `export type X = z.infer<typeof x>` restates a schema that is documented right
# above it. Requiring its own summary would produce a second copy of the truth,
# which is the thing the standard tells you not to write.
INFERRED_ALIAS_RE = re.compile(
    r"^export\s+type\s+[A-Za-z_$][\w$]*(?:<[^>]*>)?\s*=\s*z\.(?:infer|input|output)\b"
)

# Files whose only job is re-exporting or configuration; a header is enough.
TRIVIAL_NAMES = {"__init__.py", "index.ts", "index.js", "conftest.py"}


@dataclass
class Finding:
    """One missing piece of documentation, at a file and line."""

    path: str
    line: int
    symbol: str
    missing: str

    def render(self) -> str:
        """Return the one-line human form: `path:line  symbol — what is missing`."""
        return f"{self.path}:{self.line}  {self.symbol} — {self.missing}"


# --- discovery --------------------------------------------------------------


def iter_sources(roots: list[Path], stack: str | None) -> list[Path]:
    """Return every source file under `roots`, minus vendored and build dirs."""
    wanted = SOURCE_SUFFIXES
    if stack == "python":
        wanted = PY_SUFFIXES
    elif stack == "node":
        wanted = TS_SUFFIXES
    elif stack == "go":
        wanted = GO_SUFFIXES

    found: list[Path] = []
    for root in roots:
        if root.is_file():
            if root.suffix in wanted:
                found.append(root)
            continue
        for path in sorted(root.rglob("*")):
            if path.suffix not in wanted or not path.is_file():
                continue
            if SKIP_DIR_PARTS & set(path.parts):
                continue
            if path.name.startswith("."):
                continue
            found.append(path)
    return found


def is_test(path: Path) -> bool:
    """Return True for a test file, which is documented by its test names."""
    parts = set(path.parts)
    return (
        "tests" in parts
        or "test" in parts
        or path.name.startswith("test_")
        or path.name.endswith(("_test.go", ".test.ts", ".spec.ts", ".test.tsx"))
    )


def in_dirs(path: Path, dirs: tuple[str, ...]) -> bool:
    """Return True when `path` sits under one of the given repo-relative dirs."""
    posix = path.as_posix()
    return any(f"/{d}/" in f"/{posix}" for d in dirs)


# --- per-language checks ----------------------------------------------------


def check_python(path: Path, text: str) -> list[Finding]:
    """Check a Python file's module, class and function docstrings."""
    rel = path.as_posix()
    try:
        tree = ast.parse(text)
    except SyntaxError as exc:  # a file that does not parse is the gate's problem
        return [Finding(rel, exc.lineno or 1, path.name, f"does not parse: {exc.msg}")]

    findings: list[Finding] = []
    module_doc = ast.get_docstring(tree)
    if not module_doc:
        findings.append(Finding(rel, 1, path.name, "no file header"))

    if path.name in TRIVIAL_NAMES:
        return findings

    layered = in_dirs(path, LAYERED_DIRS)
    if in_dirs(path, AGENT_DIRS) and module_doc and not NODE_BLOCK_RE.search(module_doc):
        findings.append(
            Finding(rel, 1, path.name, "agent node header has no Tools/Budget/Emits block")
        )

    for node in ast.walk(tree):
        if not isinstance(
            node, (ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)
        ):
            continue
        # Private helpers are named, not documented; dunders are boilerplate
        # (`__init__`, `__repr__`) that the standard explicitly exempts.
        if node.name.startswith("_"):
            continue
        doc = ast.get_docstring(node)
        kind = "class" if isinstance(node, ast.ClassDef) else "function"
        if not doc:
            findings.append(Finding(rel, node.lineno, node.name, f"exported {kind} has no summary"))
            continue
        if (
            layered
            and kind == "function"
            and not isinstance(node, ast.ClassDef)
            and not CALLED_FROM_RE.search(doc)
            and node.col_offset == 0
        ):
            findings.append(Finding(rel, node.lineno, node.name, "no `Called from:` line"))
    return findings


def _has_block_comment_above(lines: list[str], index: int) -> bool:
    """Return True when a `/** ... */` or `//` comment sits directly above line `index`."""
    i = index - 1
    while i >= 0 and not lines[i].strip():
        i -= 1
    if i < 0:
        return False
    stripped = lines[i].strip()
    return stripped.startswith(("*/", "*", "//", "/**"))


def check_ts(path: Path, text: str) -> list[Finding]:
    """Check a TypeScript or JavaScript file's header and exported symbols."""
    rel = path.as_posix()
    lines = text.splitlines()
    findings: list[Finding] = []

    head = "\n".join(lines[:20])
    first = next((l.strip() for l in lines if l.strip()), "")
    if not first.startswith(("/**", "//", "/*")):
        findings.append(Finding(rel, 1, path.name, "no file header"))

    if path.name in TRIVIAL_NAMES:
        return findings

    layered = in_dirs(path, LAYERED_DIRS)
    if in_dirs(path, AGENT_DIRS) and not NODE_BLOCK_RE.search(head):
        findings.append(
            Finding(rel, 1, path.name, "agent node header has no Tools/Budget/Emits block")
        )

    for n, line in enumerate(lines):
        m = TS_EXPORT_RE.match(line.strip())
        if not m:
            continue
        if INFERRED_ALIAS_RE.match(line.strip()):
            continue
        name, kind = m.group("name"), m.group("kind")
        if not _has_block_comment_above(lines, n):
            findings.append(Finding(rel, n + 1, name, f"exported {kind} has no summary"))
            continue
        if layered and kind in {"function", "const"}:
            above = "\n".join(lines[max(0, n - 12) : n])
            if not CALLED_FROM_RE.search(above):
                findings.append(Finding(rel, n + 1, name, "no `Called from:` line"))
    return findings


def has_package_comment(text: str) -> bool:
    """Return True when a Go file carries the package comment above `package x`."""
    lines = text.splitlines()
    pkg = next((n for n, l in enumerate(lines) if l.startswith("package ")), None)
    return pkg is not None and pkg > 0 and lines[pkg - 1].strip().startswith("//")


def check_go_packages(sources: list[Path]) -> list[Finding]:
    """Report one finding per Go package that no file documents.

    Exactly one file per package carries the package comment, so this runs once
    over the whole scan rather than per file — otherwise an undocumented package
    reports once for every file in it.
    """
    packages: dict[Path, list[Path]] = {}
    for path in sources:
        if path.suffix in GO_SUFFIXES:
            packages.setdefault(path.parent, []).append(path)
    findings = []
    for directory, files in sorted(packages.items()):
        if any(has_package_comment(f.read_text(encoding="utf-8", errors="replace")) for f in files):
            continue
        first = sorted(files)[0]
        findings.append(
            Finding(first.as_posix(), 1, directory.name, "package has no package comment")
        )
    return findings


def check_go(path: Path, text: str) -> list[Finding]:
    """Check a Go file's package comment and exported declarations."""
    rel = path.as_posix()
    lines = text.splitlines()
    findings: list[Finding] = []

    layered = in_dirs(path, LAYERED_DIRS)
    for n, line in enumerate(lines):
        m = GO_FUNC_RE.match(line) or GO_TYPE_RE.match(line)
        if not m:
            continue
        name = m.group("name")
        above_line = lines[n - 1].strip() if n else ""
        if not above_line.startswith("//"):
            findings.append(Finding(rel, n + 1, name, "exported declaration has no comment"))
            continue
        # GoDoc renders the whole contiguous block, so the convention applies to
        # its first line; a `Called from:` line directly above the declaration is
        # the normal case and must not be read as the opening sentence.
        first = n - 1
        while first > 0 and lines[first - 1].strip().startswith("//"):
            first -= 1
        if not lines[first].strip().startswith(f"// {name}"):
            findings.append(
                Finding(rel, n + 1, name, f"comment should start with `// {name} `")
            )
        if layered and GO_FUNC_RE.match(line):
            above = "\n".join(lines[max(0, n - 12) : n])
            if not CALLED_FROM_RE.search(above):
                findings.append(Finding(rel, n + 1, name, "no `Called from:` line"))
    return findings


CHECKERS = {
    **{s: check_python for s in PY_SUFFIXES},
    **{s: check_ts for s in TS_SUFFIXES},
    **{s: check_go for s in GO_SUFFIXES},
}


# --- callers ----------------------------------------------------------------


def exported_symbols(path: Path, text: str) -> list[str]:
    """Return the exported symbol names defined in one file."""
    names: list[str] = []
    if path.suffix in PY_SUFFIXES:
        try:
            tree = ast.parse(text)
        except SyntaxError:
            return names
        for node in tree.body:
            if isinstance(node, (ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)):
                if not node.name.startswith("_"):
                    names.append(node.name)
    elif path.suffix in TS_SUFFIXES:
        names = [m.group("name") for m in (TS_EXPORT_RE.match(l.strip()) for l in text.splitlines()) if m]
    elif path.suffix in GO_SUFFIXES:
        for line in text.splitlines():
            m = GO_FUNC_RE.match(line) or GO_TYPE_RE.match(line)
            if m:
                names.append(m.group("name"))
    return sorted(set(names))


def module_name(path: Path) -> str:
    """Return the dotted module name used in a `Called from:` line."""
    parts = list(path.with_suffix("").parts)
    for prefix in ("src", "internal"):
        if prefix in parts:
            parts = parts[parts.index(prefix) + 1 :]
            break
    return ".".join(parts)


def report_callers(target: Path, roots: list[Path]) -> int:
    """Print a ready-to-paste `Called from:` line for each export in `target`."""
    text = target.read_text(encoding="utf-8", errors="replace")
    symbols = exported_symbols(target, text)
    if not symbols:
        print(f"{target}: no exported symbols")
        return 0

    sources = [p for p in iter_sources(roots, None) if p.resolve() != target.resolve()]
    bodies = {p: p.read_text(encoding="utf-8", errors="replace") for p in sources}

    for symbol in symbols:
        pattern = re.compile(rf"\b{re.escape(symbol)}\s*[\(\{{<]")
        callers = sorted(
            {module_name(p) for p, body in bodies.items() if pattern.search(body) and not is_test(p)}
        )
        if not callers:
            line = "Called from: nothing (dead code?)"
        elif len(callers) <= 3:
            line = "Called from: " + ", ".join(callers)
        else:
            line = f"Called from: {', '.join(callers[:3])}, and {len(callers) - 3} others"
        print(f"{symbol}\n    {line}")
    return 0


# --- entry point ------------------------------------------------------------


def main(argv: list[str] | None = None) -> int:
    """Run the documentation gate and return its exit code."""
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--paths", nargs="*", default=["."], help="directories to scan")
    parser.add_argument("--stack", choices=("python", "node", "go"), help="limit to one stack")
    parser.add_argument("--report", metavar="PATH", help="write findings as JSON")
    parser.add_argument("--callers", metavar="FILE", help="print Called-from lines for one file")
    parser.add_argument("--max", type=int, default=40, help="findings to print (default 40)")
    parser.add_argument(
        "--include-tests", action="store_true", help="check test files too (off by default)"
    )
    args = parser.parse_args(argv)

    roots = [Path(p) for p in args.paths]
    for root in roots:
        if not root.exists():
            print(f"docgate: {root} does not exist", file=sys.stderr)
            return 2

    if args.callers:
        target = Path(args.callers)
        if not target.exists():
            print(f"docgate: {target} does not exist", file=sys.stderr)
            return 2
        return report_callers(target, roots)

    sources = [p for p in iter_sources(roots, args.stack) if args.include_tests or not is_test(p)]
    if not sources:
        print("docgate: no source files to check", file=sys.stderr)
        return 2

    findings: list[Finding] = []
    for path in sources:
        checker = CHECKERS.get(path.suffix)
        if checker is None:
            continue
        findings.extend(checker(path, path.read_text(encoding="utf-8", errors="replace")))
    findings.extend(check_go_packages(sources))

    if args.report:
        out = Path(args.report)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(
            json.dumps(
                {"files_checked": len(sources), "findings": [asdict(f) for f in findings]},
                indent=2,
            ),
            encoding="utf-8",
        )

    if not findings:
        print(f"docgate: {len(sources)} files checked, nothing missing")
        return 0

    for finding in findings[: args.max]:
        print(finding.render())
    if len(findings) > args.max:
        print(f"... and {len(findings) - args.max} more (raise --max, or read the --report JSON)")
    print(f"\ndocgate: {len(findings)} findings across {len(sources)} files")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
