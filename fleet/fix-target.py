#!/usr/bin/env python3
"""Cible d'un !fix : PR ouverte liée à l'issue + axe (labels axe/*).
Sortie : «SLUG|BRANCH|AXE» (BRANCH vide = fallback titre ; AXE vide = pas de label d'axe).
Seules les branches fleet/* sont des cibles valides (projets de la flotte).
"""
import json
import os
import re
import subprocess
import sys

sys.path.insert(0, os.path.dirname(__file__))
from slugify import slug  # noqa: E402

repo = os.environ.get("REPO", "")
issue = os.environ.get("ISSUE_NUMBER", "")

# Axe depuis les labels de l'issue
labels = json.loads(subprocess.run(
    ["gh", "issue", "view", issue, "-R", repo, "--json", "labels"],
    capture_output=True, text=True, check=True,
).stdout)
axe = ""
for lab in labels.get("labels", []):
    if lab["name"].startswith("axe/"):
        axe = lab["name"].removeprefix("axe/")
        break

out = subprocess.run(
    ["gh", "pr", "list", "-R", repo, "--state", "open",
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
        print(f"{head.removeprefix('fleet/')}|{head}|{axe}")
        sys.exit(0)
    if f"#{issue}" in body and fallback is None:
        fallback = head
if fallback:
    print(f"{fallback.removeprefix('fleet/')}|{fallback}|{axe}")
    sys.exit(0)
print(f"{slug(os.environ.get('ISSUE_TITLE', ''))}||{axe}")