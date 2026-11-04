package web

import (
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"teleblock/internal/profile"
	"teleblock/internal/rules"
)

// testServer installs a dashboard with one known TV (samsung) and an
// injectable LAN discovery, so no test ever touches the real network.
func testServer(t *testing.T, hosts []Host, arpErr error) *Server {
	t.Helper()
	s := New(profile.New(rules.DefaultSets()), map[string]string{"192.168.1.50": "samsung"})
	s.arp = func() ([]Host, error) {
		if arpErr != nil {
			return nil, arpErr
		}
		return hosts, nil
	}
	s.hostname = func(string) string { return "" }
	return s
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func postForm(t *testing.T, h http.Handler, path, form string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// plainHTML approximates what a browser displays for the page: character
// entities decoded, typographic spaces folded to plain spaces. The tests
// assert the language the household actually sees, not the escaping bytes.
func plainHTML(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ") // no-break space
	s = strings.ReplaceAll(s, "\u202f", " ") // narrow no-break space
	s = strings.ReplaceAll(s, "\u2009", " ") // thin space
	return s
}

func TestDashboardShowsTVWithCounterInPlainLanguage(t *testing.T) {
	s := testServer(t, []Host{
		{IP: "192.168.1.50", MAC: "b8:27:eb:12:34:56"},
		{IP: "192.168.1.99", MAC: "dc:a6:32:aa:bb:cc"},
	}, nil)
	for i := 0; i < 12; i++ {
		s.counter.RecordBlocked("192.168.1.50")
	}

	rec := get(t, s, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / : got %d, want 200", rec.Code)
	}
	body := plainHTML(rec.Body.String())
	for _, want := range []string{
		"192.168.1.50",
		"b8:27:eb:12:34:56",
		"TV Samsung",
		"<strong>12</strong>",
		"tentatives de tracking bloquées aujourd'hui",
		`class="lamp hot"`,
		`value="samsung" class="chip sel"`,
		"unknown device",
		"dc:a6:32:aa:bb:cc",
		"Aucune tentative de tracking bloquée aujourd'hui",
		"1 TV sous protection",
	} {
		if !strings.Contains(body, plainHTML(want)) {
			t.Errorf("dashboard: missing %q", want)
		}
	}
}

func TestPlainBlockedWordingCoversZeroOneAndMany(t *testing.T) {
	cases := map[int]string{
		0:  "Aucune tentative de tracking bloquée aujourd'hui",
		1:  "1 tentative de tracking bloquée aujourd'hui",
		2:  "2 tentatives de tracking bloquées aujourd'hui",
		27: "27 tentatives de tracking bloquées aujourd'hui",
	}
	for n, want := range cases {
		if got := PlainBlocked(n); got != want {
			t.Errorf("PlainBlocked(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestAssignOneClickTurnsDeviceIntoTV(t *testing.T) {
	s := testServer(t, []Host{{IP: "192.168.1.99", MAC: "dc:a6:32:aa:bb:cc"}}, nil)
	rec := postForm(t, s, "/assign", "ip=192.168.1.99&brand=lg")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /assign : got %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("redirect = %q, want %q", loc, "/")
	}
	body := get(t, s, "/").Body.String()
	if !strings.Contains(body, `value="lg" class="chip sel"`) {
		t.Error("the lg brand was not kept after the assignment")
	}
	if !strings.Contains(body, "TV LG") {
		t.Error("the device should display itself as a TV after assignment")
	}
	if !strings.Contains(body, `class="lamp on"`) {
		t.Error("an assigned TV should show the green status lamp")
	}
}

func TestAssignRejectsUnknownBrand(t *testing.T) {
	s := testServer(t, nil, nil)
	rec := postForm(t, s, "/assign", "ip=192.168.1.99&brand=fridge")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /assign brand=fridge : got %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "fridge") {
		t.Error("the error must name the refused brand")
	}
	if strings.Contains(get(t, s, "/").Body.String(), "fridge") {
		t.Error("an unknown brand must not reach the panel")
	}
}

func TestAssignRejectsBadIP(t *testing.T) {
	s := testServer(t, nil, nil)
	if rec := postForm(t, s, "/assign", "ip=not-an-ip&brand=samsung"); rec.Code != http.StatusBadRequest {
		t.Errorf("POST /assign ip=not-an-ip : got %d, want 400", rec.Code)
	}
}

func TestAssignRejectsEmptyBrand(t *testing.T) {
	s := testServer(t, nil, nil)
	if rec := postForm(t, s, "/assign", "ip=192.168.1.99&brand="); rec.Code != http.StatusBadRequest {
		t.Errorf("POST /assign without brand : got %d, want 400", rec.Code)
	}
}

func TestParseArpKeepsResolvedEntriesOnly(t *testing.T) {
	const cache = `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.50     0x1         0x2         b8:27:eb:12:34:56     *        eth0
192.168.1.101    0x1         0x0         00:00:00:00:00:00     *        eth0
192.168.1.102    0x1         0x2         ff:ff:ff:ff:ff:ff     *        eth0
garbage          0x1         0x2         aa:bb:cc:dd:ee:ff     *        eth0
`
	hosts := parseArp([]byte(cache))
	if len(hosts) != 1 {
		t.Fatalf("parseArp: got %d host(s), want 1", len(hosts))
	}
	if hosts[0].IP != "192.168.1.50" || hosts[0].MAC != "b8:27:eb:12:34:56" {
		t.Errorf("parseArp: got %v, want the resolved entry", hosts[0])
	}
}

func TestDashboardSurvivesArpScanFailure(t *testing.T) {
	s := testServer(t, nil, errors.New("ARP cache unavailable"))
	rec := get(t, s, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / with failing ARP scan : got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "192.168.1.50") {
		t.Error("an already known TV must stay visible")
	}
	if strings.Contains(body, "unknown device") {
		t.Error("a failed ARP scan must not invent devices")
	}
}

func TestCounterResetsTheNextDay(t *testing.T) {
	day1 := time.Date(2025, 11, 3, 22, 0, 0, 0, time.UTC)
	day2 := day1.Add(24 * time.Hour)
	c := NewCounter()
	c.now = func() time.Time { return day1 }
	for i := 0; i < 3; i++ {
		c.RecordBlocked("192.168.1.50")
	}
	if got := c.BlockedToday("192.168.1.50"); got != 3 {
		t.Fatalf("BlockedToday on day D = %d, want 3", got)
	}
	c.now = func() time.Time { return day2 }
	if got := c.BlockedToday("192.168.1.50"); got != 0 {
		t.Errorf("the counter must reset to zero the next day, got %d", got)
	}
	c.RecordBlocked("192.168.1.50")
	if got := c.BlockedToday("192.168.1.50"); got != 1 {
		t.Errorf("BlockedToday on day D+1 = %d, want 1", got)
	}
	c.now = func() time.Time { return day1 }
	if got := c.BlockedToday("192.168.1.50"); got != 0 {
		t.Errorf("after prune, yesterday's count is forgotten, got %d", got)
	}
}

func TestCounterSeparatesDevices(t *testing.T) {
	c := NewCounter()
	c.RecordBlocked("192.168.1.50")
	c.RecordBlocked("192.168.1.50")
	c.RecordBlocked("192.168.1.99")
	if got := c.BlockedToday("192.168.1.50"); got != 2 {
		t.Errorf("counter for the TV = %d, want 2", got)
	}
	if got := c.BlockedToday("192.168.1.99"); got != 1 {
		t.Errorf("counter for the other device = %d, want 1", got)
	}
	if got := c.BlockedToday("10.0.0.1"); got != 0 {
		t.Errorf("counter for an absent device = %d, want 0", got)
	}
}

func TestAPIDevicesReturnsJSON(t *testing.T) {
	s := testServer(t, []Host{{IP: "10.1.2.3", MAC: "aa:bb:cc:00:11:22"}}, nil)
	s.counter.RecordBlocked("192.168.1.50")
	rec := get(t, s, "/api/devices")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/devices : got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var devices []struct {
		IP      string `json:"ip"`
		Brand   string `json:"brand"`
		MAC     string `json:"mac"`
		Blocked int    `json:"blocked_today"`
		Text    string `json:"blocked_today_text"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &devices); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("got %d device(s), want 2 (the TV + the machine seen via ARP)", len(devices))
	}
	type seen struct {
		brand, mac, text string
		blocked          int
	}
	byIP := make(map[string]seen)
	for _, d := range devices {
		byIP[d.IP] = seen{brand: d.Brand, mac: d.MAC, text: d.Text, blocked: d.Blocked}
	}
	tv := byIP["192.168.1.50"]
	if tv.brand != "samsung" || tv.blocked != 1 || tv.text != "1 tentative de tracking bloquée aujourd'hui" {
		t.Errorf("TV in JSON: %+v", tv)
	}
	other := byIP["10.1.2.3"]
	if other.brand != "" || other.mac != "aa:bb:cc:00:11:22" || other.blocked != 0 ||
		other.text != "Aucune tentative de tracking bloquée aujourd'hui" {
		t.Errorf("unprofiled device in JSON: %+v", other)
	}
}

func TestRoutesRejectUnknownPathAndWrongMethod(t *testing.T) {
	s := testServer(t, nil, nil)
	if rec := get(t, s, "/nope"); rec.Code != http.StatusNotFound {
		t.Errorf("GET /nope : got %d, want 404", rec.Code)
	}
	if rec := postForm(t, s, "/", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST / : got %d, want 405", rec.Code)
	}
}

func TestHostnameNamesTheDevice(t *testing.T) {
	s := testServer(t, nil, nil)
	s.hostname = func(ip string) string {
		if ip == "192.168.1.50" {
			return "salon.local"
		}
		return ""
	}
	if body := get(t, s, "/").Body.String(); !strings.Contains(body, "salon.local") {
		t.Error("the reverse-DNS name must take over as device label")
	}
}
