#!/usr/bin/env python3
"""Fail CI when total Go statement coverage is below the chosen threshold."""
from pathlib import Path
import re, sys

path = Path(sys.argv[1] if len(sys.argv) > 1 else "coverage.txt")
threshold = float(sys.argv[2] if len(sys.argv) > 2 else "70")
if not path.exists():
    raise SystemExit(f"coverage report not found: {path}")
text = path.read_text(encoding="utf-8", errors="replace")
match = re.search(r"^total:\s+\(statements\)\s+([0-9.]+)%\s*$", text, re.M)
if not match:
    raise SystemExit("could not find total statement coverage")
value = float(match.group(1))
print(f"Go statement coverage: {value:.1f}% (minimum {threshold:.1f}%)")
if value < threshold:
    raise SystemExit(1)
