// Package tests porte la scène E2E réelle de TeleBlock : le démon est
// compilé et démarré pour de vrai, et son comportement observable est
// vérifié sur de vrais sockets UDP et TCP (voir e2e_test.go).
//
// La scène est sous la contrainte de compilation « e2e » pour que la
// suite unitaire (go test ./...) reste rapide ; tests/e2e.sh la lance
// avec -tags e2e.
//
// Ce fichier n'est pas décoratif : sans lui, le paquet tests ne
// contiendrait que des fichiers exclus par la contrainte, et go vet
// ./... / go build ./... échouent avec « build constraints exclude all
// Go files in .../tests » — l'arbitre devient rouge avant même
// d'exécuter un test.
package tests
