#!/usr/bin/env python3
"""Small repository secret regression scan (not a replacement for a dedicated scanner)."""
from pathlib import Path
import re, sys

ROOT = Path(__file__).resolve().parents[1]
IGNORE_DIRS = {".git", "vendor", "node_modules"}
IGNORE_NAMES = {".env.example"}
patterns = {
    "OpenAI key": re.compile(r"\bsk-(?:proj-)?[A-Za-z0-9_-]{20,}\b"),
    "Telegram bot token": re.compile(r"\b\d{6,12}:[A-Za-z0-9_-]{25,}\b"),
    "Private key": re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    "JWT-like hardcoded token": re.compile(r"\beyJ[a-zA-Z0-9_-]{20,}\.[a-zA-Z0-9_-]{20,}\.[a-zA-Z0-9_-]{15,}\b"),
}
findings=[]
for path in ROOT.rglob("*"):
    if not path.is_file() or any(p in IGNORE_DIRS for p in path.parts) or path.name in IGNORE_NAMES:
        continue
    if path.suffix.lower() in {".zip", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2"}:
        continue
    try: text=path.read_text(encoding="utf-8")
    except Exception: continue
    for label, rx in patterns.items():
        for m in rx.finditer(text):
            line=text.count("\n",0,m.start())+1
            findings.append(f"{path.relative_to(ROOT)}:{line}: {label}")
if findings:
    print("Potential secrets found:")
    print("\n".join(findings))
    sys.exit(1)
print("PASS: no obvious committed secrets detected")
