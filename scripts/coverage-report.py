#!/usr/bin/env python3
"""Render a Markdown coverage summary from a Go coverage profile.

Usage: coverage-report.py <profile> [base-profile]

With a base profile the per-package and total deltas are included, so a pull
request that lowers coverage is visible without opening the artifact.
"""

import sys
from collections import defaultdict

MODULE = "github.com/profitify/profitify-backend/"


def summarise(path):
    """Map package -> (covered statements, total statements)."""
    covered = defaultdict(int)
    total = defaultdict(int)

    with open(path, encoding="utf-8") as fh:
        for lineno, line in enumerate(fh):
            line = line.strip()
            # The first line is the coverage mode, e.g. "mode: set".
            if not line or lineno == 0:
                continue
            # Format: <file>:<start>,<end> <numStatements> <hitCount>
            parts = line.split()
            if len(parts) < 3:
                continue
            try:
                stmts, hits = int(parts[-2]), int(parts[-1])
            except ValueError:
                continue
            pkg = line.split(":", 1)[0].rsplit("/", 1)[0]
            if pkg.startswith(MODULE):
                pkg = pkg[len(MODULE):]
            total[pkg] += stmts
            if hits > 0:
                covered[pkg] += stmts

    return {pkg: (covered[pkg], total[pkg]) for pkg in total}


def pct(covered, total):
    return 0.0 if total == 0 else (covered * 100.0) / total


def emoji(p):
    if p >= 90:
        return "🟢"
    if p >= 70:
        return "🟡"
    return "🔴"


def delta(head, base):
    """Format the change against the base branch, or '' when unchanged."""
    if base is None:
        return ""
    d = head - base
    if d > 0.05:
        return f" **(+{d:.1f})**"
    if d < -0.05:
        return f" **({d:.1f})**"
    return ""


def main():
    if len(sys.argv) < 2:
        print(__doc__.strip(), file=sys.stderr)
        return 2

    head = summarise(sys.argv[1])
    base = {}
    if len(sys.argv) > 2:
        try:
            base = summarise(sys.argv[2])
        except OSError:
            base = {}
    has_base = bool(base)

    head_covered = sum(c for c, _ in head.values())
    head_total = sum(t for _, t in head.values())
    total_pct = pct(head_covered, head_total)

    total_delta = ""
    if has_base:
        base_covered = sum(c for c, _ in base.values())
        base_total = sum(t for _, t in base.values())
        total_delta = delta(total_pct, pct(base_covered, base_total))

    out = [
        f"### {emoji(total_pct)} Test Coverage",
        "",
        f"**Total: {total_pct:.1f}%**{total_delta}",
        "",
        "| Package | Coverage | Statements |",
        "|---|---:|---:|",
    ]

    # Lowest coverage first, so packages needing attention lead the table.
    for pkg in sorted(head, key=lambda p: (pct(*head[p]), p)):
        covered, total = head[pkg]
        p = pct(covered, total)
        if pkg in base:
            d = delta(p, pct(*base[pkg]))
        elif has_base:
            d = " *(new)*"
        else:
            d = ""
        out.append(f"| `{pkg}` | {p:.1f}% {emoji(p)}{d} | {covered}/{total} |")

    footer = "🟢 ≥90% · 🟡 ≥70% · 🔴 <70%"
    if has_base:
        footer += " · deltas are against the base branch"
    out += ["", f"<sub>{footer}</sub>"]

    print("\n".join(out))
    return 0


if __name__ == "__main__":
    sys.exit(main())
