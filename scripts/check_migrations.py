#!/usr/bin/env python3
from pathlib import Path
import re, sys
root=Path(__file__).resolve().parents[1]/"migrations"
files=sorted(p.name for p in root.glob("*.sql"))
rx=re.compile(r"^(\d{6})_.+\.(up|down)\.sql$")
pairs={}
for f in files:
    m=rx.match(f)
    if not m: raise SystemExit(f"invalid migration filename: {f}")
    pairs.setdefault(m.group(1),set()).add(m.group(2))
nums=sorted(int(n) for n in pairs)
if nums and nums != list(range(nums[0], nums[-1]+1)):
    raise SystemExit(f"migration sequence has a gap: {nums}")
for n,kinds in pairs.items():
    if kinds != {"up","down"}: raise SystemExit(f"migration {n} missing up/down pair: {kinds}")
print(f"PASS: {len(pairs)} ordered migration pairs ({nums[0] if nums else 0}..{nums[-1] if nums else 0})")
