#!/usr/bin/env python3
"""Cible d'un !fix : trouve la PR ouverte liée à l'issue, en déduit slug + branche.
Sortie : «SLUG|BRANCH» si PR liée trouvée (BRANCH non vide), sinon «SLUG|» (fallback titre).
"""
import json
import os
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
for pr in json.loads(out):
    body = pr.get("body") or ""
    if f"#{issue}" in body:
        branch = pr["headRefName"]
        print(f"{branch.removeprefix('fleet/')}|{branch}")
        sys.exit(0)
print(f"{slug(os.environ.get('ISSUE_TITLE', ''))}|")