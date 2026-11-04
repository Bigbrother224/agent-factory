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
//
// Note sur la méthode : ces artefacts sont de la CONFIGURATION livrée
// (compose, Dockerfile, scripts POSIX) — pour eux, une sous-chaîne
// précise EST le contrat (le compose doit monter /var/lib/teleblock,
// l'entrypoint doit trapper EXIT…) ; les relations entre fichiers
// (ports, variables, règles) et les exécutions réelles (sh -n, compose
// config, motifs testés comportementalement) font le reste.
package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// émulation QEMU). Version de Go épinglée sur celle de go.mod — le
// test lit go.mod au lieu de coder la version en dur (revue : sinon le
// contrat ne change pas mais le test casse).
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
	gomod, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		t.Fatalf("go.mod illisible : %v", err)
	}
	m := regexp.MustCompile(`(?m)^go (\d+\.\d+)`).FindSubmatch(gomod)
	if m == nil {
		t.Fatal("go.mod : directive « go X.Y » introuvable")
	}
	assertDeployContains(t, "Dockerfile", d, "golang:"+string(m[1])+"-alpine")
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
	assertDeployContains(t, "entrypoint.sh", e, "trap cleanup EXIT INT TERM")
	if runtime.GOOS == "windows" {
		t.Skip("pas de sh sous windows")
	}
	if out, err := exec.Command("sh", "-n", filepath.Join(deployDir, "entrypoint.sh")).CombinedOutput(); err != nil {
		t.Fatalf("entrypoint.sh : erreur de syntaxe POSIX : %v\n%s", err, out)
	}
}

// Cohérence INTERNE de l'entrypoint : les trois occurrences de la règle
// (-C test, -A ajout, -D nettoyage) doivent porter exactement la même
// règle, sinon la règle installée n'est jamais celle retirée (fuite
// iptables dans l'espace réseau de l'hôte). Relation vérifiée, pas
// simple présence (revue : tests taillés sur l'implémentation).
func TestEntrypointRedirectRuleIsCoherent(t *testing.T) {
	e := readDeployFile(t, "entrypoint.sh")
	re := regexp.MustCompile(`iptables -t nat (-C|-A|-D) PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports (\S+)`)
	ms := re.FindAllStringSubmatch(e, -1)
	if len(ms) != 3 {
		t.Fatalf("entrypoint.sh : %d règles iptables REDIRECT, attendu 3 (-C, -A, -D)", len(ms))
	}
	seen := map[string]string{}
	for _, m := range ms {
		if prev, ok := seen[m[1]]; ok && prev != m[2] {
			t.Errorf("verbe %s dupliqué avec des cibles différentes", m[1])
		}
		seen[m[1]] = m[2]
		if m[2] != `"$TLS_PORT"` {
			t.Errorf("règle %s : cible %q, attendu \"$TLS_PORT\"", m[1], m[2])
		}
	}
	for _, v := range []string{"-C", "-A", "-D"} {
		if _, ok := seen[v]; !ok {
			t.Errorf("entrypoint.sh : verbe %s absent — règle non idempotente ou jamais nettoyée", v)
		}
	}
}

// Accord entrypoint ↔ compose sur le port d'écoute TLS : la règle
// iptables REDIRECT vise $TLS_PORT, le binaire écoute sur
// TELEBLOCK_TLS_PORT — même variable, même défaut (8449), sinon la
// règle renverrait les flux 443 vers un port que rien n'écoute.
func TestComposeAndEntrypointAgreeOnTLSPort(t *testing.T) {
	compose := readDeployFile(t, "docker-compose.yml")
	e := readDeployFile(t, "entrypoint.sh")
	assertDeployContains(t, "docker-compose.yml", compose, `":${TELEBLOCK_TLS_PORT:-8449}"`)
	assertDeployContains(t, "entrypoint.sh", e, `TLS_PORT="${TELEBLOCK_TLS_PORT:-8449}"`)
}

// Accord install.sh ↔ compose sur le port DNS : install.sh écrit
// TELEBLOCK_DNS_PORT dans deploy/.env, le compose l'interpole dans le
// -listen du binaire — même variable des deux côtés, sinon la pile
// écouterait sur un port différent de celui annoncé.
func TestInstallAndComposeAgreeOnDNSPort(t *testing.T) {
	install := readDeployFile(t, "install.sh")
	compose := readDeployFile(t, "docker-compose.yml")
	assertDeployContains(t, "install.sh", install, "TELEBLOCK_DNS_PORT=%s")
	assertDeployContains(t, "docker-compose.yml", compose, `":${TELEBLOCK_DNS_PORT:-53}"`)
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

// Le motif de nom Pi-hole doit être STRICT (revue : « pihole-exporter »
// matchait et déclenchait à tort repli de port et import bidon). Test
// comportemental du motif réel : les vrais noms passent, les imposteurs
// (sidecar, préfixe collé) sont rejetés.
func TestPiholeNamePatternIsStrict(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pas de sh sous windows")
	}
	const pat = `(^|[-_])pihole([-_][0-9]+)?$`
	for _, hit := range []string{"pihole", "docker-pihole", "PiHole-1", "pihole_2"} {
		if err := exec.Command("sh", "-c", "echo "+hit+" | grep -Eqi '"+pat+"'").Run(); err != nil {
			t.Errorf("%q devrait matcher le motif Pi-hole", hit)
		}
	}
	for _, miss := range []string{"pihole-exporter", "mypihole", "xpihole", "piholeweb"} {
		if err := exec.Command("sh", "-c", "echo "+miss+" | grep -Eqi '"+pat+"'").Run(); err == nil {
			t.Errorf("%q ne doit PAS matcher le motif Pi-hole (imposteur)", miss)
		}
	}
	s := readDeployFile(t, "install.sh")
	assertDeployContains(t, "install.sh", s, `PIHOLE_NAME_RE='(^|[-_])pihole([-_][0-9]+)?$'`)
	assertDeployContains(t, "install.sh", s, `grep -Eqi "$PIHOLE_NAME_RE"`)
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
