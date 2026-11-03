#!/usr/bin/env python3
"""Prochaine tranche non faite d'une roadmap : imprime la ligne, vide si finie/absente."""
import sys

path = sys.argv[1] if len(sys.argv) > 1 else ""
try:
    todo = [line.rstrip() for line in open(path) if line.startswith("- [ ]")]
except FileNotFoundError:
    todo = []
print(todo[0] if todo else "", end="")