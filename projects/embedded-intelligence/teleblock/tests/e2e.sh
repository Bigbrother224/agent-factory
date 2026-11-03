#!/bin/bash
# Scène E2E réelle de TeleBlock (barème règle 7) : le binaire est
# compilé puis démarré POUR DE VRAI, et son comportement observable est
# vérifié de bout en bout — résolution DNS par profil, sinkhole des
# domaines de télémétrie, RST ciblé sur SNI bloqué, inspection laissée
# intacte sinon, dashboard servi. Aucun mock : de vrais paquets sur de
# vrais sockets.
set -euo pipefail
cd "$(dirname "$0")/.."
exec go test -tags e2e -count=1 -timeout 300s ./tests/