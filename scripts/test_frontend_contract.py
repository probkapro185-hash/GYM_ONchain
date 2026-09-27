#!/usr/bin/env python3
"""Static regression tests for the zero-build SFEDU frontend.

These checks intentionally avoid a browser dependency, so they can run in CI and on
any developer machine with Python. Runtime behavior is covered by backend/API tests
and the optional E2E smoke script.
"""
from html.parser import HTMLParser
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
HTML = (ROOT / "frontend/index.html").read_text(encoding="utf-8")
JS = (ROOT / "frontend/assets/app.js").read_text(encoding="utf-8")
CSS = (ROOT / "frontend/assets/app.css").read_text(encoding="utf-8")

class Parser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.ids = []
        self.scripts = []
        self.links = []
    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if attrs.get("id"):
            self.ids.append(attrs["id"])
        if tag == "script" and attrs.get("src"):
            self.scripts.append(attrs["src"])
        if tag == "link" and attrs.get("href"):
            self.links.append(attrs["href"])

p = Parser(); p.feed(HTML)
failures = []

def check(condition, message):
    if not condition:
        failures.append(message)

check(len(p.ids) == len(set(p.ids)), "duplicate HTML ids")
check("/assets/app.js" in p.scripts, "app.js is not linked")
check("/assets/app.css" in p.links, "app.css is not linked")
check('<meta name="theme-color"' in HTML, "theme-color metadata missing")
check('<meta name="color-scheme"' in HTML, "color-scheme metadata missing")
check('<meta name="referrer" content="no-referrer"' in HTML, "strict referrer policy missing")

# Auth must be a normal login flow; end users must never paste JWT/backend URLs.
check("POST('/auth/login'" in JS, "login endpoint is not wired")
check("GET('/users/me')" in JS, "session validation endpoint is not wired")
check("Bearer token" not in HTML and "Backend URL" not in HTML, "developer-only auth controls leaked into UI")
check(JS.count("localStorage.setItem('sfedu_auth'") == 1, "auth state must be persisted only through the centralized saveAuth helper")

# Current backend contracts that have regressed in older frontend revisions.
required_routes = [
    "/users/me", "/trainers", "/schedule/requests", "/shop/products",
    "/shop/purchase", "/finance/me/payments", "/crm/progress",
    "/crm/notifications", "/crm/tasks", "/crm/dashboard", "/shop/staff-purchase",
    "/auth/forgot-password", "/auth/activate", "/auth/reset-password",
    "/users/invite", "/resend-activation", "/trainers/",
]
for route in required_routes:
    check(route in JS, f"missing frontend API contract: {route}")
check("/finance/me/topup" not in JS, "unsafe legacy client self-topup endpoint returned")
check("app-pass" not in HTML and "app-pass" not in JS, "staff approval still asks for a client password")
check("password_setup_required" in JS, "pending account activation state is not rendered")
check("history.replaceState" in JS, "one-time account token is not removed from browser URL")
check("ct-photo-file" in HTML and 'type="file"' in HTML, "trainer photo file picker missing")
check("compressTrainerPhoto" in JS and "image/webp" in JS, "trainer photo browser compression missing")
check("FormData()" in JS and "'/photo'" in JS, "trainer photo multipart upload missing")
check("320,0.72" in JS and "96*1024" in JS, "trainer photo size guard missing")
check("ct-photo-url" not in HTML and "ct-photo-url" not in JS, "legacy trainer photo URL input returned")

# AI implementation contract must remain unchanged while the rest of CRM evolves.
check("const body={message:msg}" in JS and "POST('/ai/chat',body)" in JS, "AI RAG request contract changed")
check("messages:[{role:" not in JS and "user_id:state.user" not in JS, "legacy AI request payload returned")

# Security/UX regression guards.
check("function esc(" in JS, "HTML escaping helper missing")
check("function saveAuth(" in JS and "function clearAuth(" in JS, "central auth persistence helpers missing")
check("prefers-reduced-motion" in CSS, "reduced-motion accessibility support missing")
check(":focus-visible" in CSS, "keyboard focus styling missing")
check("@media(max-width:768px)" in CSS, "mobile breakpoint missing")
check(CSS.count("{") == CSS.count("}"), "unbalanced CSS braces")

# Every static getElementById target should exist. Dynamic prefixes are excluded.
static_ids = set(re.findall(r"getElementById\(['\"]([^'\"]+)['\"]\)", JS))
dynamic_ids = set(re.findall(r"\.id=['\"]([^'\"]+)['\"]", JS)) | set(re.findall(r"id=[\"']([^\"']+)[\"']", JS))
missing = sorted(static_ids - set(p.ids) - dynamic_ids)
check(not missing, "JS references missing static DOM ids: " + ", ".join(missing[:20]))

if failures:
    print("FAIL")
    for item in failures:
        print(" -", item)
    sys.exit(1)
print(f"PASS: frontend contract, {len(p.ids)} DOM ids, {len(required_routes)} critical API routes")
