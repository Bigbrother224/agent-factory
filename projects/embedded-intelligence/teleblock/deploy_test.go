package main

// Tests des artefacts de déploiement — tranche T6 (roadmap).
//
// La livraison T6 est de la configuration pure ; la discipline TDD de
// l'usine la couvre par des tests de chargement : chaque règle métier du
// déploiement est verrouillée par une assertion sur le fichier livré.
//
//   - compose : port DNS 53 UDP+TCP surchargeable (Pi-hole existant),
//     état persistant, NET_ADMIN pour la redirection iptables, build
//     depuis le Dockerfile du dépôt ;
//   - Dockerfile : multi-arch (build sur $BUILDPLATFORM, cible
//     $TARGETOS/$TARGETARCH), binaire statique, runtime avec iptables,
//     DNS exposé, volume d'état ;
//   - install.sh : POSIX strict pipable (`curl | sh`), prérequis docker,
//     détection d'un Pi-hole existant, import de sa gravity, repli de
//     port 53 -> 5353, démarrage de la pile compose.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// deployProjectDir : répertoire du présent test, indépendant du cwd.
var deployProjectDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}()

func readDeployFile(t *testing.T, relPath string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(deployProjectDir, relPath))
	if err != nil {
		t.Fatalf("artefact de déploiement manquant : %v", err)
	}
	return string(b)
}

func assertDeployContains(t *testing.T, artifact, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s : exigence non vérifiée, %q absent", artifact, needle)
	}
}

func lastDockerfileStage(t *testing.T, dockerfile string) string {
	t.Helper()
	stages := strings.Split(dockerfile, "FROM ")
	if len(stages) < 2 {
		t.Fatal("Dockerfile sans étage runtime")
	}
	return stages[len(stages)-1]
}

// Un résolveur DNS répond en UDP et TCP sur 53 ; quand un Pi-hole occupe
// déjà le 53 de l'hôte, le port hôte se surcharge via TELEBLOCK_DNS_PORT
// (install.sh écrit deploy/.env pour lui).
func TestComposeBindsPort53UDPAndTCP(t *testing.T) {
	compose := readDeployFile(t, filepath.Join("deploy", "docker-compose.yml"))
	for _, proto := range []string{"udp", "tcp"} {
		assertDeployContains(t, "docker-compose.yml", compose,
			fmt.Sprintf(`"${TELEBLOCK_DNS_PORT:-53}:53/%s"`, proto))
	}
}

// L'état d'apprentissage (T3) doit survivre aux recréations de conteneur :
// montage persistant vers /var/lib/teleblock, chemin surchargeable.
func TestComposePersistsState(t *testing.T) {
	compose := readDeployFile(t, filepath.Join("deploy", "docker-compose.yml"))
	assertDeployContains(t, "docker-compose.yml", compose,
		"${TELEBLOCK_STATE:-./data}:/var/lib/teleblock")
}

// La redirection iptables des flux TCP 443 bloqués (T1) exige NET_ADMIN
// dans le conteneur.
func TestComposeGrantsNETAdminForTCPRedirect(t *testing.T) {
	compose := readDeployFile(t, filepath.Join("deploy", "docker-compose.yml"))
	assertDeployContains(t, "docker-compose.yml", compose, "- NET_ADMIN")
}

// Le compose construit l'image depuis le Dockerfile du dépôt, pas d'une
// image publiée qui ne serait pas celle du code courant.
func TestComposeBuildsFromDeployDockerfile(t *testing.T) {
	compose := readDeployFile(t, filepath.Join("deploy", "docker-compose.yml"))
	assertDeployContains(t, "docker-compose.yml", compose, "dockerfile: deploy/Dockerfile")
}

// Image multi-arch : étage de build sur la plateforme de l'hôte, cross-
// compile vers $TARGETOS/$TARGETARCH (amd64 + arm64/Pi, sans émulation QEMU).
func TestDockerfileIsMultiArch(t *testing.T) {
	d := readDeployFile(t, filepath.Join("deploy", "Dockerfile"))
	for _, want := range []string{
		"FROM --platform=$BUILDPLATFORM",
		"ARG TARGETOS",
		"ARG TARGETARCH",
		"GOOS=$TARGETOS",
		"GOARCH=$TARGETARCH",
	} {
		assertDeployContains(t, "deploy/Dockerfile", d, want)
	}
}

// Binaire statique (CGO désactivé) : l'étage runtime reste minimal.
func TestDockerfileBuildsStaticBinary(t *testing.T) {
	d := readDeployFile(t, filepath.Join("deploy", "Dockerfile"))
	assertDeployContains(t, "deploy/Dockerfile", d, "CGO_ENABLED=0")
}

// La redirection TCP (proxy SNI + reset) repose sur iptables : l'étage
// runtime (dernier FROM) doit l'embarquer, sinon les blocages partent en
// erreur à l'exécution.
func TestDockerfileRuntimeShipsIptables(t *testing.T) {
	runtimeStage := lastDockerfileStage(t, readDeployFile(t, filepath.Join("deploy", "Dockerfile")))
	assertDeployContains(t, "deploy/Dockerfile (runtime)", runtimeStage, "apk add")
	assertDeployContains(t, "deploy/Dockerfile (runtime)", runtimeStage, "iptables")
}

// L'image déclare le même canal DNS que le compose.
func TestDockerfileExposesDNS(t *testing.T) {
	d := readDeployFile(t, filepath.Join("deploy", "Dockerfile"))
	assertDeployContains(t, "deploy/Dockerfile", d, "53/udp")
	assertDeployContains(t, "deploy/Dockerfile", d, "53/tcp")
}

// L'image et le compose partagent le même chemin d'état persistant.
func TestDockerfileDeclaresStateVolume(t *testing.T) {
	d := readDeployFile(t, filepath.Join("deploy", "Dockerfile"))
	assertDeployContains(t, "deploy/Dockerfile", d, "VOLUME /var/lib/teleblock")
}

// `curl | sh` : shebang POSIX et mode strict (échouer tôt, rien à moitié fait).
func TestInstallScriptIsStrictPOSIXShell(t *testing.T) {
	s := readDeployFile(t, filepath.Join("deploy", "install.sh"))
	if !strings.HasPrefix(s, "#!/bin/sh") {
		t.Errorf("install.sh : shebang #!/bin/sh requis pour curl|sh")
	}
	assertDeployContains(t, "deploy/install.sh", s, "set -eu")
}

// Cas d'erreur attendu : sans docker (ou sans plugin compose), le script
// refuse d'installer au lieu de quitter à moitié configuré.
func TestInstallScriptRequiresDocker(t *testing.T) {
	s := readDeployFile(t, filepath.Join("deploy", "install.sh"))
	assertDeployContains(t, "deploy/install.sh", s, "command -v docker")
	assertDeployContains(t, "deploy/install.sh", s, "docker compose version")
}

// « Déjà un Pi-hole » : détection (install natif ou conteneur) puis export
// un-domaine-par-ligne de sa gravity vers l'état TeleBlock.
func TestInstallScriptImportsExistingPiholeGravity(t *testing.T) {
	s := readDeployFile(t, filepath.Join("deploy", "install.sh"))
	for _, want := range []string{
		"/etc/pihole/gravity.db",
		"SELECT domain FROM gravity",
		"docker exec",
	} {
		assertDeployContains(t, "deploy/install.sh", s, want)
	}
}

// Conflit de port : si un Pi-hole est détecté sur :53, TeleBlock bascule
// sur :5353 et l'écrit dans deploy/.env (sinon le bind du compose échoue).
// Le choix reste surchargeable via TELEBLOCK_DNS_PORT.
func TestInstallScriptAvoidsPort53Conflict(t *testing.T) {
	s := readDeployFile(t, filepath.Join("deploy", "install.sh"))
	assertDeployContains(t, "deploy/install.sh", s, "5353")
	assertDeployContains(t, "deploy/install.sh", s, "TELEBLOCK_DNS_PORT=")
}

// Garde-fou d'exécution : la syntaxe doit passer `sh -n` (POSIX strict).
func TestInstallScriptSyntaxIsValid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pas de sh sous windows")
	}
	cmd := exec.Command("sh", "-n", filepath.Join(deployProjectDir, "deploy", "install.sh"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install.sh : erreur de syntaxe POSIX : %v\n%s", err, out)
	}
}

// Un script pipé depuis curl ne doit jamais lire stdin, sinon il avale le
// reste du flux téléchargé.
func TestInstallScriptNeverReadsStdin(t *testing.T) {
	s := readDeployFile(t, filepath.Join("deploy", "install.sh"))
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "read" || strings.HasPrefix(trimmed, "read ") {
			t.Errorf("install.sh : lecture de stdin interdite pour curl|sh : %q", line)
		}
	}
}

// Fin de course : le script démarre bien la pile compose en détaché.
func TestInstallScriptStartsComposeStack(t *testing.T) {
	s := readDeployFile(t, filepath.Join("deploy", "install.sh"))
	assertDeployContains(t, "deploy/install.sh", s,
		"docker compose -f deploy/docker-compose.yml up")
	assertDeployContains(t, "deploy/install.sh", s, "up -d")
}
