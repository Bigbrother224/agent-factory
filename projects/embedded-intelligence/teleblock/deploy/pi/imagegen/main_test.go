package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// generateForTest produit une charge utile dans un répertoire temporaire.
// Le binaire ARM64 est optionnel : la configuration seule est testable sans
// cross-compile.
func generateForTest(t *testing.T, opts Options) (string, Manifest) {
	t.Helper()
	if opts.Out == "" {
		opts.Out = t.TempDir()
	}
	m, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate(%+v): erreur inattendue: %v", opts, err)
	}
	return opts.Out, m
}

// readRootfs lit un fichier de l'image via son chemin absolu sur la carte.
func readRootfs(t *testing.T, out, imagePath string) string {
	t.Helper()
	data, err := os.ReadFile(hostPath(filepath.Join(out, RootfsDir), imagePath))
	if err != nil {
		t.Fatalf("lecture de %s dans l'image: %v", imagePath, err)
	}
	return string(data)
}

// readRepoFile lit un fichier du dépôt : go test exécute chaque test avec le
// répertoire du package comme CWD, soit deploy/pi/imagegen.
func readRepoFile(t *testing.T, repoPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(repoPath)))
	if err != nil {
		t.Fatalf("lecture de %s: %v", repoPath, err)
	}
	return string(data)
}

func TestGenerateWritesSystemdUnitThatRunsTheGateway(t *testing.T) {
	out, _ := generateForTest(t, Options{Version: "test"})

	unit := readRootfs(t, out, UnitPath)
	for _, want := range []string{
		"ExecStart=/usr/local/bin/teleblock",
		"Wants=network-online.target",
		"After=network-online.target",
		"Restart=on-failure",
		"StateDirectory=teleblock",
		"EnvironmentFile=-/etc/teleblock/teleblock.env",
		// Port 53 sans root : le Pi reste une passerelle DNS durcie.
		"AmbientCapabilities=CAP_NET_BIND_SERVICE",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("unité systemd: %q absent\n---\n%s", want, unit)
		}
	}
}

func TestGenerateMakesTeleblockLocalReachableAtFirstBoot(t *testing.T) {
	out, m := generateForTest(t, Options{})

	if got := readRootfs(t, out, HostnamePath); got != "teleblock\n" {
		t.Errorf("nom d'hôte = %q, attendu %q : c'est lui qui publie teleblock.local", got, "teleblock\n")
	}
	if got := readRootfs(t, out, HostsPath); !strings.Contains(got, "127.0.1.1 teleblock.local teleblock") {
		t.Errorf("/etc/hosts doit résoudre teleblock.local localement, obtenu:\n%s", got)
	}
	if m.Host != "teleblock" {
		t.Errorf("manifeste: hostname = %q, attendu teleblock", m.Host)
	}
	// avahi publie le nom en mDNS : sans lui, aucune TV ne trouve
	// teleblock.local avant d'avoir déjà le DNS du Pi.
	link := filepath.Join(out, RootfsDir, "etc/systemd/system/multi-user.target.wants/avahi-daemon.service")
	if _, err := os.Readlink(link); err != nil {
		t.Errorf("avahi-daemon doit être activé au premier boot: %v", err)
	}
}

func TestGenerateEnablesTeleblockServiceWithoutBootScript(t *testing.T) {
	out, _ := generateForTest(t, Options{})

	link := filepath.Join(out, RootfsDir, "etc/systemd/system/multi-user.target.wants/teleblock.service")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("lien d'activation absent: %v", err)
	}
	if target != UnitPath {
		t.Errorf("cible du lien = %q, attendu %q", target, UnitPath)
	}
}

func TestGenerateFreesPort53FromSystemdResolved(t *testing.T) {
	out, _ := generateForTest(t, Options{})

	conf := readRootfs(t, out, ResolvedPath)
	if !strings.Contains(conf, "DNSStubListener=no") {
		t.Errorf("systemd-resolved occuperait le port 53 et empêcherait TeleBlock de démarrer:\n%s", conf)
	}
}

func TestGenerateInstallsTheARM64BinaryAsExecutable(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "teleblock-linux-arm64")
	if err := os.WriteFile(bin, []byte("\x7fELF faux binaire arm64"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _ := generateForTest(t, Options{Binary: bin})

	installed := filepath.Join(out, RootfsDir, "usr/local/bin/teleblock")
	data, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("binaire absent de l'image: %v", err)
	}
	if string(data) != "\x7fELF faux binaire arm64" {
		t.Errorf("binaire installé différent de la source")
	}
	info, err := os.Stat(installed)
	if err != nil {
		t.Fatal(err)
	}
	// 0755 quel que soit le umask de la machine de build.
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("mode du binaire = %04o, attendu 0755", perm)
	}
}

func TestGenerateWithoutBinaryLeavesNoExecutable(t *testing.T) {
	out, _ := generateForTest(t, Options{})

	installed := filepath.Join(out, RootfsDir, "usr/local/bin/teleblock")
	if _, err := os.Stat(installed); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("sans --binary, aucun binaire ne doit être posé (err=%v)", err)
	}
}

func TestGenerateFailsWhenTheBinaryIsMissing(t *testing.T) {
	_, err := Generate(Options{Out: t.TempDir(), Binary: filepath.Join(t.TempDir(), "absent")})
	if err == nil {
		t.Fatal("un binaire introuvable doit échouer, sinon l'image part sans passerelle")
	}
	if !strings.Contains(err.Error(), "binaire") {
		t.Errorf("message peu explicite: %v", err)
	}
}

func TestGenerateRequiresAnOutputDirectory(t *testing.T) {
	if _, err := Generate(Options{}); err == nil {
		t.Fatal("--out manquant doit échouer plutôt que d'écrire dans le répertoire courant")
	}
}

func TestManifestInventoriesEveryFileWithItsDigest(t *testing.T) {
	out, m := generateForTest(t, Options{Version: "1.2.3"})

	if m.Version != "1.2.3" {
		t.Errorf("version du manifeste = %q, attendue 1.2.3", m.Version)
	}

	want := []string{
		"/etc/hostname",
		"/etc/hosts",
		"/etc/systemd/resolved.conf.d/teleblock.conf",
		"/etc/systemd/system/multi-user.target.wants/avahi-daemon.service",
		"/etc/systemd/system/multi-user.target.wants/teleblock.service",
		"/etc/systemd/system/teleblock.service",
	}
	got := make([]string, 0, len(m.Files))
	for _, e := range m.Files {
		got = append(got, e.Path)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inventaire = %v, attendu %v", got, want)
	}

	// Le manifeste doit être vrai : chaque digest est recalculé sur la carte.
	for _, e := range m.Files {
		if e.Kind != "file" {
			continue
		}
		p := hostPath(filepath.Join(out, RootfsDir), e.Path)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("lecture %s: %v", e.Path, err)
		}
		sum := sha256.Sum256(data)
		if e.SHA256 != hex.EncodeToString(sum[:]) {
			t.Errorf("%s: digest %s incohérent avec le fichier", e.Path, e.SHA256)
		}
		if e.Size != int64(len(data)) {
			t.Errorf("%s: taille %d incohérente (%d)", e.Path, e.Size, len(data))
		}
	}

	// Le manifeste est écrit à côté du rootfs, prêt pour l'audit post-flash.
	var onDisk Manifest
	raw, err := os.ReadFile(filepath.Join(out, ManifestName))
	if err != nil {
		t.Fatalf("manifeste %s absent: %v", ManifestName, err)
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("manifeste illisible: %v", err)
	}
	if !reflect.DeepEqual(onDisk, m) {
		t.Errorf("manifeste sur disque != manifeste retourné par Generate")
	}
}

func TestGenerateIsIdempotent(t *testing.T) {
	out := t.TempDir()
	first, err := Generate(Options{Out: out, Version: "même"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(Options{Out: out, Version: "même"})
	if err != nil {
		t.Fatalf("relancer la génération doit fonctionner (make pi-image deux fois): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("deux générations identiques divergent:\n%v\n%v", first, second)
	}
}

func TestBuildScriptCrossCompilesAStaticARM64Binary(t *testing.T) {
	script := readRepoFile(t, "deploy/pi/build-image.sh")

	for _, want := range []string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=arm64", "go build", "imagegen"} {
		if !strings.Contains(script, want) {
			t.Errorf("build-image.sh: %q absent — l'image ne serait pas statique ARM64", want)
		}
	}
}

func TestMakefileOffersPiImageAndARM64Targets(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")

	for _, want := range []string{"pi-image:", "arm64:", "GOARCH=arm64", "CGO_ENABLED=0"} {
		if !strings.Contains(makefile, want) {
			t.Errorf("Makefile: %q absent", want)
		}
	}
}

func TestReadmePiDocumentsFlashableInstall(t *testing.T) {
	readme := readRepoFile(t, "deploy/pi/README-Pi.md")

	for _, want := range []string{"Raspberry Pi Imager", "teleblock.local", "make pi-image", "port 53"} {
		if !strings.Contains(readme, want) {
			t.Errorf("README-Pi.md: %q absent", want)
		}
	}
}
