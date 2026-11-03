// Package deploy : tests des artefacts de déploiement — tranche T6
// (roadmap), durcis par les retours de revue (les tests de sous-chaînes
// doivent verrouiller des CONTRATS de comportement, pas refléter
// l'implémentation).
//
//   - compose : réseau hôte (le trafic 443 des TV doit atteindre
//     l'inspecteur SNI — un bridge ne publierait que le 53), DNS
//     écouté sur ${TELEBLOCK_DNS_PORT}, état persistant, NET_ADMIN ;
//   - Dockerfile : multi-arch, version de Go épinglée, binaire
//     statique, runtime avec iptables, entrypoint embarqué ;
//   - entrypoint : règle iptables idempotente, nettoyage à l'arrêt ;
//   - install.sh : POSIX strict pipable, détection Pi-hole, import de
//     la gravity dans l'état VU par le conteneur, repli de port validé ;
//   - .dockerignore : le contexte de build n'embarque ni .git ni état.
package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// deployDir : répertoire du présent test (deploy/), indépendant du cwd.
var deployDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}()

// projectDir : racine du projet (contexte de build du compose).
var projectDir = filepath.Dir(deployDir)

func readDeployFile(t *testing.T, relPath string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(deployDir, relPath))
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

// Le trafic 443 des TV doit entrer dans l'espace réseau du conteneur
// pour que la redirection iptables (T1) l'attrape : un bridge ne publie
// que le 53 et laisse l'inspection SNI inopérante (bug réel relevé par
// la revue). Host network = l'espace de l'hôte, là où passent les flux
// LAN ; les ports publiés (ports:) sont alors inutiles et absents.
func TestComposeRunsOnHostNetwork(t *testing.T) {
	compose := readDeployFile(t, "docker-compose.yml")
	assertDeployContains(t, "docker-compose.yml", compose, "network_mode: host")
	if strings.Contains(compose, "ports:") {
		t.Errorf("docker-compose.yml : ports: inutile et trompeur avec network_mode host")
	}
}

// Le port DNS écouté par le binaire suit TELEBLOCK_DNS_PORT (53 par
// défaut, 5353/5354 écrits dans deploy/.env par install.sh quand un
// Pi-hole tient le 53) : le compose doit le passer au binaire.
func TestComposeExposesDNSViaCommand(t *testing.T) {
	compose := readDeployFile(t, "docker-compose.yml")
	assertDeployContains(t, "docker-compose.yml", compose, `":${TELEBLOCK_DNS_PORT:-53}"`)
}

// L'état d'apprentissage (T3) doit survivre aux recréations de
// conteneur : montage persistant vers /var/lib/teleblock, chemin
// surchargeable, et le binaire y écrit bien son observe.json.
func TestComposePersistsState(t *testing.T) {
	compose := readDeployFile(t, "docker-compose.yml")
	assertDeployContains(t, "docker-compose.yml", compose,
		"${TELEBLOCK_STATE:-./data}:/var/lib/teleblock")
	assertDeployContains(t, "docker-compose.yml", compose,
		"/var/lib/teleblock/observe.json")
}

// La redirection iptables des flux TCP 443 bloqués (T1) exige NET_ADMIN
// dans le conteneur.
func TestComposeGrantsNETAdminForTCPRedirect(t *testing.T) {
	compose := readDeployFile(t, "docker-compose.yml")
	assertDeployContains(t, "docker-compose.yml", compose, "- NET_ADMIN")
}

// Le compose construit l'image depuis le Dockerfile du dépôt, pas d'une
// image publiée qui ne serait pas celle du code courant.
func TestComposeBuildsFromDeployDockerfile(t *testing.T) {
	compose := readDeployFile(t, "docker-compose.yml")
	assertDeployContains(t, "docker-compose.yml", compose, "dockerfile: deploy/Dockerfile")
}

// Validation réelle du compose quand l'outillage est présent (revue :
// des assertions de sous-chaînes ne prouvent pas que le YAML est
// exploitable) ; saut propre sinon — docker compose config est
// hors-ligne, aucun démon requis.
func TestComposeConfigValidates(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker absent : validation compose reportée à l'environnement qui l'a")
	}
	if err := exec.Command("docker", "compose", "version").Run(); err != nil {
		t.Skip("plugin docker compose absent")
	}
	cmd := exec.Command("docker", "compose", "-f", filepath.Join(deployDir, "docker-compose.yml"),
		"--project-directory", deployDir, "config", "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docker compose config -q : %v\n%s", err, out)
	}
}

// Image multi-arch : étage de build sur la plateforme de l'hôte,
// cross-compile vers $TARGETOS/$TARGETARCH (amd64 + arm64/Pi, sans
// émulation QEMU). Version de Go épinglée sur go.mod (revue : un tag
// flottant casse la reproductibilité des builds).
func TestDockerfileIsMultiArchWithPinnedGo(t *testing.T) {
	d := readDeployFile(t, "Dockerfile")
	for _, want := range []string{
		"FROM --platform=$BUILDPLATFORM",
		"ARG TARGETOS",
		"ARG TARGETARCH",
		"GOOS=$TARGETOS",
		"GOARCH=$TARGETARCH",
	} {
		assertDeployContains(t, "Dockerfile", d, want)
	}
	if !strings.Contains(d, "golang:1.21-alpine") {
		t.Errorf("Dockerfile : version de Go non épinglée sur celle de go.mod (golang:1.21-alpine)")
	}
}

// Binaire statique (CGO désactivé) : l'étage runtime reste minimal.
func TestDockerfileBuildsStaticBinary(t *testing.T) {
	d := readDeployFile(t, "Dockerfile")
	assertDeployContains(t, "Dockerfile", d, "CGO_ENABLED=0")
}

// La redirection TCP (proxy SNI + reset) repose sur iptables : l'étage
// runtime (dernier FROM) doit l'embarquer, sinon les blocages partent
// en erreur à l'exécution.
func TestDockerfileRuntimeShipsIptables(t *testing.T) {
	d := readDeployFile(t, "Dockerfile")
	stages := strings.Split(d, "FROM ")
	runtimeStage := stages[len(stages)-1]
	assertDeployContains(t, "Dockerfile (runtime)", runtimeStage, "apk add")
	assertDeployContains(t, "Dockerfile (runtime)", runtimeStage, "iptables")
}

// L'image déclare le même canal DNS que le compose.
func TestDockerfileExposesDNS(t *testing.T) {
	d := readDeployFile(t, "Dockerfile")
	assertDeployContains(t, "Dockerfile", d, "53/udp")
	assertDeployContains(t, "Dockerfile", d, "53/tcp")
}

// L'image et le compose partagent le même chemin d'état persistant.
func TestDockerfileDeclaresStateVolume(t *testing.T) {
	d := readDeployFile(t, "Dockerfile")
	assertDeployContains(t, "Dockerfile", d, "VOLUME /var/lib/teleblock")
}

// Le conteneur démarre via l'entrypoint qui applique la règle iptables
// (c'est la couche déploiement qui l'installe, jamais le package Go) :
// idempotente (-C puis -A), retirée à l'arrêt (trap), port aligné sur
// le -tls-listen du compose.
func TestEntrypointAppliesRedirectRuleIdempotently(t *testing.T) {
	e := readDeployFile(t, "entrypoint.sh")
	assertDeployContains(t, "entrypoint.sh", e,
		"iptables -t nat -C PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports")
	assertDeployContains(t, "entrypoint.sh", e,
		"iptables -t nat -D PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports")
	assertDeployContains(t, "entrypoint.sh", e, "trap cleanup TERM INT")
	if runtime.GOOS == "windows" {
		t.Skip("pas de sh sous windows")
	}
	if out, err := exec.Command("sh", "-n", filepath.Join(deployDir, "entrypoint.sh")).CombinedOutput(); err != nil {
		t.Fatalf("entrypoint.sh : erreur de syntaxe POSIX : %v\n%s", err, out)
	}
}

// `curl | sh` : shebang POSIX et mode strict (échouer tôt, rien à
// moitié fait) ; sans lecture de stdin, sinon le script avale le reste
// du flux téléchargé.
func TestInstallScriptIsStrictPOSIXShellWithoutStdin(t *testing.T) {
	s := readDeployFile(t, "install.sh")
	if !strings.HasPrefix(s, "#!/bin/sh") {
		t.Errorf("install.sh : shebang #!/bin/sh requis pour curl|sh")
	}
	assertDeployContains(t, "install.sh", s, "set -eu")
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "read" || strings.HasPrefix(trimmed, "read ") {
			t.Errorf("install.sh : lecture de stdin interdite pour curl|sh : %q", line)
		}
	}
}

// Cas d'erreur attendu : sans docker (ou sans plugin compose), le
// script refuse d'installer au lieu de quitter à moitié configuré.
func TestInstallScriptRequiresDocker(t *testing.T) {
	s := readDeployFile(t, "install.sh")
	assertDeployContains(t, "install.sh", s, "command -v docker")
	assertDeployContains(t, "install.sh", s, "docker compose version")
}

// « Déjà un Pi-hole » : détection (install natif ou conteneur — liste
// des conteneurs lue UNE fois, revue : deux docker ps peuvent désigner
// des conteneurs différents) puis export un-domaine-par-ligne de sa
// gravity vers l'état TeleBlock.
func TestInstallScriptImportsExistingPiholeGravity(t *testing.T) {
	s := readDeployFile(t, "install.sh")
	for _, want := range []string{
		"/etc/pihole/gravity.db",
		"SELECT domain FROM gravity",
		"docker exec",
		"RUNNING_CONTAINERS=$(docker ps",
	} {
		assertDeployContains(t, "install.sh", s, want)
	}
}

// Contrat état install.sh ↔ compose : la gravity est importée sous
// $TELEBLOCK_STATE, que l'installateur exporte — sans quoi le conteneur
// monterait un autre répertoire que celui de l'import et la
// fonctionnalité « déjà un Pi-hole » serait inopérante (bug réel relevé
// par la revue).
func TestInstallScriptExportsCoherentStatePath(t *testing.T) {
	s := readDeployFile(t, "install.sh")
	assertDeployContains(t, "install.sh", s,
		`GRAVITY_OUT="$STATE_DIR/pihole-gravity.txt"`)
	assertDeployContains(t, "install.sh", s,
		`export TELEBLOCK_STATE="$STATE_DIR"`)
	assertDeployContains(t, "install.sh", s,
		`STATE_DIR="${TELEBLOCK_STATE:-$PWD/deploy/data}"`)
}

// set -eu rend toute variable non définie fatale AVANT le message
// d'erreur : TELEBLOCK_REPO doit être lu sous garde ${…:-} (bug réel
// relevé par la revue : le cas principal mourait en « parameter not
// set » au lieu d'expliquer le remède).
func TestInstallScriptGuardsUnsetRepoVariable(t *testing.T) {
	s := readDeployFile(t, "install.sh")
	assertDeployContains(t, "install.sh", s, `[ -n "${TELEBLOCK_REPO:-}" ]`)
	if strings.Contains(s, `[ -n "$TELEBLOCK_REPO" ]`) {
		t.Errorf("install.sh : $TELEBLOCK_REPO non gardé — meurt sous set -u avant le message d'aide")
	}
}

// Conflit de port : si un Pi-hole est détecté sur :53, TeleBlock
// bascule sur :5353 — validé comme réellement LIBRE avant d'écrire
// deploy/.env (revue : un 5353 déjà occupé ferait échouer le compose
// up), puis :5354, sinon refus explicite. Le test du port ne dépend
// pas de nc : /proc/net/tcp d'abord (Linux, zéro dépendance).
// Le choix reste surchargeable via TELEBLOCK_DNS_PORT.
func TestInstallScriptAvoidsPort53Conflict(t *testing.T) {
	s := readDeployFile(t, "install.sh")
	assertDeployContains(t, "install.sh", s, "5353")
	assertDeployContains(t, "install.sh", s, "TELEBLOCK_DNS_PORT=")
	assertDeployContains(t, "install.sh", s, "port_free 5353")
	assertDeployContains(t, "install.sh", s, "/proc/net/tcp")
}

// Le contexte de build (racine du projet) exclut .git et l'état local :
// ni fuite de dépôt, ni image gonflée (revue : COPY . . non filtré).
func TestDockerignoreExcludesVCSAndState(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(projectDir, ".dockerignore"))
	if err != nil {
		t.Fatal(".dockerignore manquant à la racine du projet")
	}
	for _, want := range []string{".git", "observe.json"} {
		if !strings.Contains(string(b), want) {
			t.Errorf(".dockerignore : %q devrait être exclu du contexte de build", want)
		}
	}
}

// Garde-fou d'exécution : la syntaxe doit passer `sh -n` (POSIX strict).
func TestInstallScriptSyntaxIsValid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pas de sh sous windows")
	}
	if out, err := exec.Command("sh", "-n", filepath.Join(deployDir, "install.sh")).CombinedOutput(); err != nil {
		t.Fatalf("install.sh : erreur de syntaxe POSIX : %v\n%s", err, out)
	}
}

// Fin de course : le script démarre bien la pile compose en détaché.
func TestInstallScriptStartsComposeStack(t *testing.T) {
	s := readDeployFile(t, "install.sh")
	assertDeployContains(t, "install.sh", s,
		"docker compose -f deploy/docker-compose.yml up")
	assertDeployContains(t, "install.sh", s, "up -d")
}
