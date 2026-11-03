//go:build e2e

// Scène E2E réelle de TeleBlock : le binaire est compilé et démarré
// pour de vrai, puis son comportement observable est vérifié de bout en
// bout. Aucun mock : de vrais paquets UDP pour le DNS, de vrais
// ClientHello TLS sur de vrais sockets TCP, le vrai dashboard HTTP.
//
// Le scénario joue le rôle d'une TV Samsung déclarée sur le routeur :
//   - ses requêtes DNS de télémétrie sont sinkholées (0.0.0.0) ;
//   - son streaming passe (résolution amont réelle) ;
//   - son flux TLS vers un domaine de télémétrie reçoit un RST ciblé ;
//   - son flux TLS vers un domaine de streaming est inspecté mais
//     jamais bloqué ;
//   - son dashboard répond.
package e2e

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	dnsAddr  = "127.0.0.1:5311"
	tlsAddr  = "127.0.0.1:8449"
	webAddr  = "127.0.0.1:8093"
	upstream = "1.1.1.1:53"

	telemetryDomain = "samsungacr.com" // Block dans le set samsung
	streamDomain    = "netflix.com"    // Allow dans le set samsung
)

// TestTeleBlockEndToEnd compile et démarre le vrai binaire, puis
// vérifie le comportement observable complet. Les sous-tests partagent
// le même processus : c'est un scénario, pas des unités isolées.
func TestTeleBlockEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("scène E2E : ignorée en mode court")
	}
	dir := t.TempDir()
	root, err := filepath.Abs(filepath.Join(".", ".."))
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "teleblock")

	// 1. Le produit compile en un vrai binaire.
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	// 2. Le produit démarre pour de vrai : DNS + inspection TLS +
	//    dashboard, avec la TV Samsung du scénario (127.0.0.1).
	//    Pas de -observe-state : le mode apprentissage suspend tout
	//    blocage pendant sa période d'observation (~1 semaine), ce qui
	//    contredit les attentes de la scène ; il est couvert par les
	//    tests unitaires (internal/observe, internal/profile). Le drapeau
	//    est passé VIDE car son défaut (observe.json) active l'observation
	//    — ici on vérifie le pare-feu tel qu'il s'applique une fois le
	//    profil de la marque en vigueur.
	logPath := filepath.Join(dir, "teleblock.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	proc := exec.Command(bin,
		"-listen", dnsAddr,
		"-upstream", upstream,
		"-device", "127.0.0.1=samsung",
		"-tls-listen", tlsAddr,
		"-web-listen", webAddr,
		"-observe-state", "")
	proc.Stdout = logFile
	proc.Stderr = logFile
	if err := proc.Start(); err != nil {
		t.Fatalf("démarrage teleblock: %v", err)
	}
	// dead est fermé à la mort du processus : le recevoir reste
	// possible autant de fois que nécessaire — l'étape 9 et le cleanup
	// attendent tous deux cette mort, un simple canal tamponné ne
	// pourrait être consommé qu'une fois.
	var waitErr error
	dead := make(chan struct{})
	go func() {
		waitErr = proc.Wait()
		close(dead)
	}()
	t.Cleanup(func() {
		proc.Process.Kill()
		<-dead
	})

	// 3. Prêt quand le dashboard répond.
	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := http.Get("http://" + webAddr + "/")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			break
		}
		select {
		case <-dead:
			t.Fatalf("teleblock est mort avant d'être prêt : %v (log: %s)", waitErr, readLog(t, logPath))
		case <-time.After(200 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Fatalf("teleblock jamais prêt (log: %s)", readLog(t, logPath))
		}
	}

	// 4. DNS : la télémétrie de la TV est sinkholée.
	t.Run("dns_sinkhole", func(t *testing.T) {
		ips := dnsA(t, telemetryDomain)
		if len(ips) != 1 || !ips[0].Equal(net.IPv4zero) {
			t.Fatalf("%s devrait être sinkholé vers 0.0.0.0, reçu %v", telemetryDomain, ips)
		}
	})

	// 5. DNS : le streaming passe (résolution amont réelle).
	t.Run("dns_passthrough", func(t *testing.T) {
		ips := dnsA(t, "example.com")
		if len(ips) == 0 {
			t.Fatal("example.com devrait être résolu en amont — aucune réponse A")
		}
		for _, ip := range ips {
			if ip.Equal(net.IPv4zero) {
				t.Fatalf("example.com ne doit jamais être sinkholé, reçu %v", ips)
			}
		}
	})

	// 6. TLS : un ClientHello vers la télémétrie reçoit un RST ciblé.
	t.Run("tls_reset_sur_sni_bloque", func(t *testing.T) {
		err := tlsHandshake(t, telemetryDomain)
		if err == nil {
			t.Fatalf("%s : le handshake aurait dû être coupé par RST, il a réussi", telemetryDomain)
		}
		if !strings.Contains(err.Error(), "connection reset") && !strings.Contains(err.Error(), "ECONNRESET") {
			t.Fatalf("%s : attendu un RST (connection reset), reçu %v", telemetryDomain, err)
		}
	})

	// 7. TLS : un ClientHello vers le streaming est inspecté mais
	//    jamais bloqué — le verdict Block est réservé à la télémétrie.
	t.Run("tls_inspecte_sans_blocage", func(t *testing.T) {
		err := tlsHandshake(t, streamDomain)
		_ = err // le relayage dépend du réseau ; on n'exige rien ici
		log := readLog(t, logPath)
		if !strings.Contains(log, "sni="+streamDomain) {
			t.Fatalf("%s : le ClientHello aurait dû être inspecté (log: %s)", streamDomain, log)
		}
		if strings.Contains(log, "blocked SNI "+streamDomain) {
			t.Fatalf("%s ne doit JAMAIS être bloqué — c'est du streaming", streamDomain)
		}
	})

	// 8. Dashboard : la TV répond sur son panneau local.
	t.Run("dashboard", func(t *testing.T) {
		resp, err := http.Get("http://" + webAddr + "/")
		if err != nil {
			t.Fatalf("dashboard injoignable : %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("dashboard : statut %d, attendu 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Fatal("dashboard : réponse vide")
		}
	})

	// 9. Le produit termine proprement quand on l'arrête.
	proc.Process.Signal(os.Interrupt)
	select {
	case <-dead:
	case <-time.After(5 * time.Second):
		t.Fatalf("teleblock ne s'est pas arrêté proprement (log: %s)", readLog(t, logPath))
	}
}

// tlsHandshake ouvre un vrai flux TLS vers l'inspecteur avec le SNI
// donné et retourne l'éventuelle erreur de handshake.
func tlsHandshake(t *testing.T, serverName string) error {
	t.Helper()
	conn, err := net.DialTimeout("tcp", tlsAddr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("connexion à l'inspecteur: %w", err)
	}
	defer conn.Close()
	tlsConn := tls.Client(conn, &tls.Config{ServerName: serverName})
	if err := tlsConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	return tlsConn.Handshake()
}

// dnsA envoie une vraie requête UDP au résolveur de la scène et
// retourne les enregistrements A de la réponse.
func dnsA(t *testing.T, name string) []net.IP {
	t.Helper()
	conn, err := net.DialTimeout("udp", dnsAddr, 3*time.Second)
	if err != nil {
		t.Fatalf("UDP vers %s: %v", dnsAddr, err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(dnsQuery(name)); err != nil {
		t.Fatalf("envoi requête DNS: %v", err)
	}
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("lecture réponse DNS pour %s: %v", name, err)
	}
	return parseA(buf[:n])
}

// dnsA et dnsQuery parlent le vrai format DNS sur le fil.
func dnsQuery(name string) []byte {
	q := make([]byte, 0, 512)
	q = append(q, 0xab, 0xcd)                        // identifiant
	q = append(q, 0x01, 0x00)                        // récursif
	q = append(q, 0, 1, 0, 0, 0, 0, 0, 0)            // 1 question
	for _, label := range strings.Split(name, ".") { // QNAME
		q = append(q, byte(len(label)))
		q = append(q, label...)
	}
	q = append(q, 0)
	q = append(q, 0, 1) // type A
	q = append(q, 0, 1) // classe IN
	return q
}

// parseA extrait les adresses IPv4 de la section réponses ; les noms
// compressés (pointeurs 0xc0) et les autres types sont sautés.
func parseA(resp []byte) []net.IP {
	if len(resp) < 12 {
		return nil
	}
	qdcount := int(binary.BigEndian.Uint16(resp[4:6]))
	ancount := int(binary.BigEndian.Uint16(resp[6:8]))
	off := 12
	skipName := func() bool {
		for {
			if off >= len(resp) {
				return false
			}
			l := int(resp[off])
			if l&0xc0 == 0xc0 { // pointeur de compression
				off += 2
				return true
			}
			off++
			if l == 0 {
				return true
			}
			off += l
		}
	}
	for i := 0; i < qdcount; i++ {
		if !skipName() || off+4 > len(resp) {
			return nil
		}
		off += 4
	}
	var ips []net.IP
	for i := 0; i < ancount; i++ {
		if !skipName() || off+10 > len(resp) {
			return ips
		}
		typ := binary.BigEndian.Uint16(resp[off : off+2])
		rdlen := int(binary.BigEndian.Uint16(resp[off+8 : off+10]))
		off += 10
		if typ == 1 && rdlen == 4 && off+4 <= len(resp) {
			ips = append(ips, net.IP(append([]byte(nil), resp[off:off+4]...)))
		}
		off += rdlen
	}
	return ips
}

// readLog lit le journal du produit démarré (première et dernière
// moitié : assez pour diagnostiquer sans noyer la sortie).
func readLog(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return "(log illisible)"
	}
	s := string(b)
	if len(s) > 2000 {
		s = s[:1000] + "\n[…]\n" + s[len(s)-1000:]
	}
	return s
}
