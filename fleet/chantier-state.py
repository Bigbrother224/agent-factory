#!/usr/bin/env python3
"""État du chantier : trouve l'issue approuvée non shippée et son axe.
Sortie : «NUM|AXE|TITLE» ou vide si aucun chantier actif (l'usine dort)."""
import json
import os
import subprocess

out = subprocess.run(
    ["gh", "issue", "list", "-R", os.environ.get("REPO", ""),
     "--label", "idea/approved", "--state", "open", "--limit", "10",
     "--json", "number,title,labels"],
    capture_output=True, text=True, check=True,
).stdout
for issue in json.loads(out):
    labs = [l["name"] for l in issue["labels"]]
    if any(l.startswith("axe/") for l in labs) and "shipped" not in labs:
        axe = next(l[4:] for l in labs if l.startswith("axe/"))
        print(f"{issue['number']}|{axe}|{issue['title']}")
        break