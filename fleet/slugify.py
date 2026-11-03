#!/usr/bin/env python3
"""Slug d'un titre → nom court de projet."""
import re
import sys
import unicodedata


def slug(title: str) -> str:
    t = unicodedata.normalize("NFKD", title).encode("ascii", "ignore").decode()
    s = re.sub(r"[^a-z0-9]+", "-", t.lower()).strip("-")
    # Slug court : les 3 premiers segments du titre, max 24 caractères.
    words = s.split("-")
    return "-".join(words[:3])[:24].strip("-") or "projet"


if __name__ == "__main__":
    print(slug(sys.argv[1] if len(sys.argv) > 1 else ""))