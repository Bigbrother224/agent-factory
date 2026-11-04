// Command imagegen assemble la charge utile de l'image Raspberry Pi de
// TeleBlock (tranche T7).
//
// Il n'écrit jamais de carte SD : il produit une arborescence rootfs/
// déterministe — unité systemd, nom d'hôte « teleblock » publié en mDNS,
// libération du port 53 pour systemd-resolved, binaire ARM64 statique,
// manifeste SHA-256 — que deploy/pi/build-image.sh archive et pose sur la
// carte.
package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// payloadFS embarque la configuration posée dans l'image : une seule source
// de vérité, donc aucun décalage possible entre le dépôt et une carte flashée.
//
//go:embed files/teleblock.service files/resolved-teleblock.conf files/hosts
var payloadFS embed.FS

// Chemins absolus dans l'image, exportés pour que tests/e2e.sh vérifie
// exactement ce que l'utilisateur trouvera sur la carte.
const (
	RootfsDir	= "rootfs"
	BinaryPath	= "/usr/local/bin/teleblock"
	UnitPath	= "/etc/systemd/system/teleblock.service"
	HostnamePath	= "/etc/hostname"
	HostsPath	= "/etc/hosts"
	ResolvedPath	= "/etc/systemd/resolved.conf.d/teleblock.conf"
	WantsDir	= "/etc/systemd/system/multi-user.target.wants"
	ManifestName	= "payload.json"
	Hostname	= "teleblock"
)

// Options décrit une génération de charge utile.
type Options struct {
	// Out est le répertoire de sortie (créé si absent).
	Out	string
	// Binary est le binaire TeleBlock linux/arm64 à poser dans l'image.
	// Vide : la configuration seule est générée (tests, revue de code).
	Binary	string
	// Version est inscrite dans le manifeste.
	Version	string
}

// Entry décrit un fichier (ou un lien) posé dans l'image.
type Entry struct {
	Path	string	`json:"path"`
	Kind	string	`json:"kind"`
	Mode	string	`json:"mode,omitempty"`
	Target	string	`json:"target,omitempty"`
	Size	int64	`json:"size,omitempty"`
	SHA256	string	`json:"sha256,omitempty"`
}

// Manifest inventorie ce qui a été posé : une carte se vérifie sans être montée.
type Manifest struct {
	Version	string	`json:"version"`
	Host	string	`json:"hostname"`
	Files	[]Entry	`json:"files"`
}

// Generate écrit la charge utile dans opts.Out et retourne son manifeste.
//
// Il est volontairement idempotent : relancer `make pi-image` sur une sortie
// existante doit produire exactement la même carte, jamais une erreur.
func Generate(opts Options) (Manifest, error) {
	if opts.Out == "" {
		return Manifest{}, errors.New("imagegen: répertoire de sortie manquant (--out)")
	}
	if opts.Binary != "" {
		if _, err := os.Stat(opts.Binary); err != nil {
			return Manifest{}, fmt.Errorf("imagegen: binaire ARM64 introuvable: %w", err)
		}
	}

	root := filepath.Join(opts.Out, RootfsDir)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Manifest{}, fmt.Errorf("imagegen: créer %s: %w", root, err)
	}

	// 1. Configuration embarquée : le système de fichiers de l'image.
	embedded := [][2]string{
		{"files/teleblock.service", UnitPath},
		{"files/resolved-teleblock.conf", ResolvedPath},
		{"files/hosts", HostsPath},
	}
	for _, f := range embedded {
		data, err := payloadFS.ReadFile(f[0])
		if err != nil {
			return Manifest{}, fmt.Errorf("imagegen: charge embarquée %s: %w", f[0], err)
		}
		if err := writeFile(root, f[1], data, 0o644); err != nil {
			return Manifest{}, err
		}
	}

	// 2. Nom d'hôte : c'est lui, publié par avahi, qui rend teleblock.local
	// joignable dès le premier boot, sans rien configurer sur les TV.
	if err := writeFile(root, HostnamePath, []byte(Hostname+"\n"), 0o644); err != nil {
		return Manifest{}, err
	}

	// 3. Activation : deux liens suffisent pour que systemd démarre les
	// services au boot suivant — aucun script d'amorçage à maintenir.
	for _, unit := range []string{UnitPath, "/etc/systemd/system/avahi-daemon.service"} {
		if err := writeSymlink(root, WantsDir+"/"+filepath.Base(unit), unit); err != nil {
			return Manifest{}, err
		}
	}

	// 4. Binaire : facultatif pour que la génération de configuration reste
	// testable sans cross-compile ; build-image.sh le passe toujours.
	if opts.Binary != "" {
		data, err := os.ReadFile(opts.Binary)
		if err != nil {
			return Manifest{}, fmt.Errorf("imagegen: lecture du binaire %s: %w", opts.Binary, err)
		}
		if err := writeFile(root, BinaryPath, data, 0o755); err != nil {
			return Manifest{}, err
		}
	}

	// 5. Manifeste : on inventorie ce qui a réellement été posé, digest
	// compris, pour pouvoir auditer une carte sans la monter.
	entries, err := inventory(root)
	if err != nil {
		return Manifest{}, err
	}
	m := Manifest{Version: opts.Version, Host: Hostname, Files: entries}
	buf, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return Manifest{}, fmt.Errorf("imagegen: encoder le manifeste: %w", err)
	}
	buf = append(buf, '\n')
	if err := os.WriteFile(filepath.Join(opts.Out, ManifestName), buf, 0o644); err != nil {
		return Manifest{}, fmt.Errorf("imagegen: écrire %s: %w", ManifestName, err)
	}
	return m, nil
}

// hostPath traduit un chemin absolu de l'image en chemin sur la machine de
// build : « /etc/hostname » → <root>/etc/hostname.
func hostPath(root, imagePath string) string {
	return filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(imagePath, "/")))
}

func writeFile(root, imagePath string, data []byte, mode os.FileMode) error {
	full := hostPath(root, imagePath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("imagegen: créer %s: %w", filepath.Dir(imagePath), err)
	}
	if err := os.WriteFile(full, data, mode); err != nil {
		return fmt.Errorf("imagegen: écrire %s: %w", imagePath, err)
	}
	// os.WriteFile n'applique le mode qu'à la création et le umask peut le
	// réduire : on force le mode pour que la carte soit reproductible.
	if err := os.Chmod(full, mode); err != nil {
		return fmt.Errorf("imagegen: mode de %s: %w", imagePath, err)
	}
	return nil
}

func writeSymlink(root, imagePath, target string) error {
	full := hostPath(root, imagePath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("imagegen: créer %s: %w", filepath.Dir(imagePath), err)
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("imagegen: remplacer %s: %w", imagePath, err)
	}
	if err := os.Symlink(target, full); err != nil {
		return fmt.Errorf("imagegen: lier %s → %s: %w", imagePath, target, err)
	}
	return nil
}

// inventory liste récursivement la charge utile, triée par chemin : deux
// générations identiques produisent un manifeste identique octet pour octet.
func inventory(root string) ([]Entry, error) {
	var entries []Entry
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel == "." || d.IsDir() {
			return nil
		}
		abs := "/" + filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			entries = append(entries, Entry{Path: abs, Kind: "symlink", Target: target})
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, Entry{
			Path:   abs,
			Kind:   "file",
			Mode:   fmt.Sprintf("%04o", info.Mode().Perm()),
			Size:   int64(len(data)),
			SHA256: hex.EncodeToString(sum[:]),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("imagegen: inventaire de %s: %w", root, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func main() {
	out := flag.String("out", "dist/pi-image", "répertoire de sortie de la charge utile")
	binary := flag.String("binary", "", "binaire TeleBlock linux/arm64 à poser dans l'image")
	version := flag.String("version", "dev", "version inscrite dans payload.json")
	flag.Parse()

	m, err := Generate(Options{Out: *out, Binary: *binary, Version: *version})
	if err != nil {
		fmt.Fprintf(os.Stderr, "imagegen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("imagegen: %d fichiers dans %s/%s, manifeste %s\n", len(m.Files), *out, RootfsDir, ManifestName)
}
