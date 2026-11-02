#!/usr/bin/env python3
"""Slug du titre d'issue → nom de projet (identique à scout-daily.sh)."""
import re
import sys
import unicodedata

title = sys.argv[1] if len(sys.argv) > 1 else ""
t = unicodedata.normalize("NFKD", title).encode("ascii", "ignore").decode()
s = re.sub(r"[^a-z0-9]+", "-", t.lower()).strip("-")
print((s[:40].rstrip("-")) or "projet")