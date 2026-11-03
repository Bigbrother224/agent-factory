#!/bin/bash
# Lance le dashboard de l'usine : serveur + ouverture navigateur
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
pkill -f "dashboard.js" 2>/dev/null || true
node "$DIR/dashboard.js" &
sleep 1.5
open "http://localhost:7777"
wait