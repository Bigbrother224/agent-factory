#!/usr/bin/env python3
"""Cible d'un !fix : trouve la PR ouverte liée à l'issue, en déduit slug + branche.
Sortie : «SLUG|BRANCH» si PR liée trouvée (BRANCH non vide), sinon «SLUG|» (fallback titre).
Seules les branches fleet/* sont des cibles valides (projets de la flotte).
"""
import json
import os
import re
import subprocess
import sys

sys.path.insert(0, os.path.dirname(__file__))
from slugify import slug  # noqa: E402

issue = os.environ.get("ISSUE_NUMBER", "")
out = subprocess.run(
    ["gh", "pr", "list", "-R", os.environ.get("REPO", ""), "--state", "open",
     "--json", "number,headRefName,body", "--limit", "30"],
    capture_output=True, text=True, check=True,
).stdout
fallback = None
for pr in json.loads(out):
    body = (pr.get("body") or "").lower()
    head = pr["headRefName"]
    if not head.startswith("fleet/") or not re.fullmatch(r"fleet/[a-z0-9-]+", head):
        continue
    if f"issue #{issue}" in body:
        print(f"{head.removeprefix('fleet/')}|{head}")
        sys.exit(0)
    if f"#{issue}" in body and fallback is None:
        fallback = head
if fallback:
    print(f"{fallback.removeprefix('fleet/')}|{fallback}")
    sys.exit(0)
print(f"{slug(os.environ.get('ISSUE_TITLE', ''))}|")