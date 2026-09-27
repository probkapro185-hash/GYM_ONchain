#!/usr/bin/env python3
from html.parser import HTMLParser
from pathlib import Path
import re, sys

root=Path(__file__).resolve().parents[1]
html=(root/'frontend/index.html').read_text(encoding='utf-8')
js=(root/'frontend/assets/app.js').read_text(encoding='utf-8')
css=(root/'frontend/assets/app.css').read_text(encoding='utf-8')
class P(HTMLParser):
    def __init__(self): super().__init__(); self.ids=[]
    def handle_starttag(self,tag,attrs):
        for k,v in attrs:
            if k=='id' and v: self.ids.append(v)
p=P(); p.feed(html)
if len(p.ids)!=len(set(p.ids)):
    dup=sorted({x for x in p.ids if p.ids.count(x)>1})
    raise SystemExit(f'duplicate HTML ids: {dup}')
required=['/assets/app.css','/assets/app.js']
for x in required:
    if x not in html: raise SystemExit(f'missing asset {x}')
for token in ['/crm/dashboard','/crm/tasks','/crm/progress','/crm/notifications','/shop/staff-purchase','/auth/forgot-password','/auth/activate','/auth/reset-password','/users/invite','/resend-activation']:
    if token not in js: raise SystemExit(f'missing CRM+ API integration {token}')
# AI contract should remain the RAG contract from the previous frontend.
if "POST('/ai/chat',body)" not in js or "const body={message:msg}" not in js:
    raise SystemExit('AI chat contract unexpectedly changed')
if 'messages:[{role:' in js or 'user_id:state.user' in js:
    raise SystemExit('legacy AI payload found')
if 'app-pass' in html or 'app-pass' in js:
    raise SystemExit('staff approval still asks for a client password')
if 'ct-photo-file' not in html or 'compressTrainerPhoto' not in js or "image/webp" not in js:
    raise SystemExit('trainer local photo upload/compression is missing')
if 'ct-photo-url' in html or 'ct-photo-url' in js:
    raise SystemExit('legacy trainer photo URL input returned')
if '<span class="nav-icon">📊</span>Dashboard' in html or '<span class="nav-icon">✅</span>Мои задачи' in html:
    raise SystemExit('dashboard/tasks still use emoji navigation icons')
for token in ['staff-profile-edit-btn','staff-profile-save-btn','staff-profile-cancel-btn']:
    if token not in html: raise SystemExit(f'missing staff profile control {token}')
for token in ['function setStaffProfileEditMode','function enableStaffProfileEdit','function cancelStaffProfileEdit','input.readOnly=!editable']:
    if token not in js: raise SystemExit(f'missing protected admin profile behavior: {token}')
# quick structural sanity checks
if css.count('{') != css.count('}'):
    raise SystemExit('CSS brace mismatch')
print(f'PASS: {len(p.ids)} unique HTML ids; CRM+ frontend wiring present; AI contract unchanged')
