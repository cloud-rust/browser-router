package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLocalState(t *testing.T, udd, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(udd, "Local State"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func testConfig(t *testing.T, udd string) Config {
	t.Helper()
	cfg, err := parseConfig([]byte(strings.ReplaceAll(`{
  "rules": [
    {"name": "development", "regex": ["^https?://(localhost|127\\.0\\.0\\.1|\\[::1\\]|[^/?#]+\\.localhost)(:\\d+)?([/?#]|$)"],
     "target": {"browser": "firefox"}},
    {"name": "admin", "match": ["*portal.azure.com*globex*"], "target": {"browser": "vivaldi", "profile": "Globex Admin", "user_data_dir": "UDD"}},
    {"name": "acme", "match": ["*acme*"], "target": {"browser": "vivaldi", "profile": "Acme", "user_data_dir": "UDD"}},
    {"name": "spotify", "match": ["*spotify.com*"], "target": {"browser": "app", "app": "Spotify"}},
    {"name": "notion", "match": ["https://notion.so*"], "target": {"browser": "app", "app": "Notion"}}
  ],
  "fallback": {"browser": "safari"}
}`, "UDD", filepath.ToSlash(udd))))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestDecide(t *testing.T) {
	udd := t.TempDir()
	// With a byte order mark, as PowerShell 5 writes it.
	writeLocalState(t, udd, "\xef\xbb\xbf"+`{"profile":{"info_cache":{"Profile 1":{"name":"Acme"},"Profile 2":{"name":"Globex Admin"}}}}`)
	cfg := testConfig(t, udd)

	cases := []struct{ url, rule, browser, profile string }{
		{"http://localhost:3000/x", "development", "firefox", ""},
		{"http://127.0.0.1:8080", "development", "firefox", ""},
		{"http://[::1]:5000/", "development", "firefox", ""},
		{"http://app.localhost:3000/", "development", "firefox", ""},
		{"https://localhost.evil.com/", "fallback", "safari", ""},
		{"https://portal.azure.com/#@ACME.onmicrosoft.com", "acme", "vivaldi", "Profile 1"},
		{"https://portal.azure.com/#@globex.onmicrosoft.com", "admin", "vivaldi", "Profile 2"},
		{"https://open.spotify.com/track/1", "spotify", "app", ""},
		{"https://notion.so/page", "notion", "app", ""},
		{"https://example.org/", "fallback", "safari", ""},
	}
	for _, c := range cases {
		d := decide(cfg, c.url)
		if d.Rule != c.rule || d.Target.Browser != c.browser || d.Target.Profile != c.profile {
			t.Errorf("%s: got %s/%s/%s, want %s/%s/%s", c.url, d.Rule, d.Target.Browser, d.Target.Profile, c.rule, c.browser, c.profile)
		}
	}
}

func TestUnknownProfileFallsBack(t *testing.T) {
	cfg := testConfig(t, t.TempDir()) // no Local State, so "Acme" cannot be resolved
	d := decide(cfg, "https://acme.sharepoint.com/")
	if d.Rule != "fallback" || d.Target.Browser != "safari" || d.Note == "" {
		t.Errorf("got %+v, want fallback with note", d)
	}
}

func TestGlob(t *testing.T) {
	cases := []struct {
		glob, url string
		want      bool
	}{
		{"*acme*", "https://ACME.sharepoint.com/x", true},
		{"*acme*", "https://example.org/", false},
		{"localhost*", "http://localhost:3000/", true}, // no scheme in the pattern: any scheme
		{"localhost*", "http://notlocalhost/", false},
		{"https://notion.so*", "https://notion.so/page", true},
		{"https://notion.so*", "http://notion.so/page", false},
		{"*notion.com*/oauth*", "https://www.notion.com/x/oauth?a=b", true},
		{"*.local", "http://printer.local", true},
		{"a.b", "https://aXb", false}, // the dot is literal
		{"*?q=1*", "https://x/?q=1", true},
		{"*login.microsoftonline.com/organizations/oauth2/v2.0/authorize*client_id=04b0*",
			"https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize?client_id=04b0&x", true},
	}
	for _, c := range cases {
		if got := globRegexp(c.glob).MatchString(c.url); got != c.want {
			t.Errorf("%q on %q: got %v, want %v", c.glob, c.url, got, c.want)
		}
	}
}

func TestParseConfigErrors(t *testing.T) {
	for _, body := range []string{
		`{"rules": [{"name": "x", "match": ["*"], "target": {"browser": "vivaldi"}}]}`,
		`{"rules": [{"name": "x", "target": {"browser": "firefox"}}]}`,
		`{"rules": [{"name": "x", "regex": ["("], "target": {"browser": "firefox"}}]}`,
		`{"rules": [], "fallback": {"browser": "netscape"}}`,
		`{"rules": [], "fallback": {"browser": "vivaldi", "profile": "A"}}`,
		`{"rules": [], "unknown": 1}`,
	} {
		if _, err := parseConfig([]byte(body)); err == nil {
			t.Errorf("no error for %s", body)
		}
	}
	cfg, err := parseConfig([]byte(`{"rules": []}`))
	if err != nil || cfg.Fallback.Browser == "" {
		t.Errorf("missing fallback: got %+v, %v", cfg.Fallback, err)
	}
}

func TestHostOf(t *testing.T) {
	if got := hostOf("https://login.example.com/cb?code=secret"); got != "https://login.example.com" {
		t.Errorf("got %q", got)
	}
	if got := hostOf("mailto:a@b.c"); got != "mailto:" {
		t.Errorf("got %q", got)
	}
}

func TestDecideForClient(t *testing.T) {
	udd := t.TempDir()
	writeLocalState(t, udd, `{"profile":{"info_cache":{"Profile 1":{"name":"Acme"},"Profile 2":{"name":"Globex"}}}}`)
	cfg := testConfig(t, udd)
	cfg.Clients = map[string]Target{
		"acme":   {Browser: "vivaldi", Profile: "Acme", UserDataDir: udd},
		"globex": {Browser: "vivaldi", Profile: "Globex", UserDataDir: udd},
		"gone":   {Browser: "vivaldi", Profile: "Gone", UserDataDir: udd},
	}
	if err := compile(&cfg); err != nil {
		t.Fatal(err)
	}
	// The Azure CLI sign-in carries no client hint; the client of the call decides.
	login := "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize?client_id=04b07795"
	cases := []struct{ client, url, rule, profile string }{
		{"globex", login, "client globex", "Profile 2"},
		{"acme", login, "client acme", "Profile 1"},
		{"acme", "http://localhost:3000/", "client acme", "Profile 1"}, // the client wins over the rules
		{"", "https://acme.sharepoint.com/", "acme", "Profile 1"},
		{"unknown", "https://example.org/", "fallback", ""}, // unknown client: the rules, never a guess
		{"gone", "https://acme.sharepoint.com/", "acme", "Profile 1"},
	}
	for _, c := range cases {
		d := decideFor(cfg, c.client, c.url)
		if d.Rule != c.rule || d.Target.Profile != c.profile {
			t.Errorf("%s %s: got %s/%s, want %s/%s", c.client, c.url, d.Rule, d.Target.Profile, c.rule, c.profile)
		}
		if (c.client == "unknown" || c.client == "gone") && d.Note == "" {
			t.Errorf("%s: no note", c.client)
		}
	}
}

func TestClientTargetsAreChecked(t *testing.T) {
	if _, err := parseConfig([]byte(`{"rules": [], "clients": {"acme": {"browser": "vivaldi"}}}`)); err == nil {
		t.Error("client without Vivaldi profile accepted")
	}
}

func TestURLs(t *testing.T) {
	udd := t.TempDir()
	writeLocalState(t, udd, `{"profile":{"info_cache":{"Profile 1":{"name":"Acme"},"Profile 2":{"name":"Globex"},"Profile 3":{"name":"Globex Admin"}}}}`)
	cfg, err := parseConfig([]byte(strings.ReplaceAll(`{
  "rules": [
    {"name": "acme", "match": ["*acme*"], "urls": ["github.com/acme", "acme.example"],
     "target": {"browser": "vivaldi", "profile": "Acme", "user_data_dir": "UDD"}},
    {"name": "globex admin", "urls": ["https://admin.globex.atlassian.net/"],
     "target": {"browser": "vivaldi", "profile": "Globex Admin", "user_data_dir": "UDD"}},
    {"name": "globex", "urls": ["*.globex.atlassian.net", "*.globex.com", "github.com/Globex"],
     "target": {"browser": "vivaldi", "profile": "Globex", "user_data_dir": "UDD"}}
  ],
  "fallback": {"browser": "safari"}
}`, "UDD", filepath.ToSlash(udd))))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ url, rule string }{
		// A client's name in a page of another client's wiki no longer moves the link.
		{"https://globex.atlassian.net/wiki/spaces/X/pages/1/Acme+Workshop", "globex"},
		{"https://admin.globex.atlassian.net/x", "globex admin"}, // the exact host beats *.
		{"https://www.globex.com/", "globex"},
		{"https://globex.com", "globex"},
		{"https://notglobex.com/", "fallback"},
		{"https://github.com/acme/repo/issues?q=globex", "acme"},
		{"https://github.com/ACME", "acme"},
		{"https://github.com/acme-corp/x", "acme"}, // no urls entry fits: the match *acme* still does
		{"https://github.com/globex/x", "globex"},
		{"https://github.com/other/acme", "acme"}, // via match
		{"https://github.com/other/x", "fallback"},
		{"https://ACME.example:8443/x", "acme"},
		{"ftp://globex.com/", "fallback"},
	}
	for _, c := range cases {
		if d := decide(cfg, c.url); d.Rule != c.rule {
			t.Errorf("%s: got %s, want %s", c.url, d.Rule, c.rule)
		}
	}
}

func TestURLErrors(t *testing.T) {
	for _, body := range []string{
		`{"rules": [{"name": "x", "urls": ["*acme*"], "target": {"browser": "firefox"}}]}`,
		`{"rules": [{"name": "x", "urls": ["acme.com:8080"], "target": {"browser": "firefox"}}]}`,
		`{"rules": [{"name": "x", "urls": ["/path"], "target": {"browser": "firefox"}}]}`,
		`{"rules": [{"name": "x", "urls": ["a.com/x*"], "target": {"browser": "firefox"}}]}`,
		`{"rules": [{"name": "x", "urls": ["a.com"], "target": {"browser": "firefox"}},
		            {"name": "y", "urls": ["A.com/"], "target": {"browser": "safari"}}]}`,
	} {
		if _, err := parseConfig([]byte(body)); err == nil {
			t.Errorf("no error for %s", body)
		}
	}
}

func TestActivation(t *testing.T) {
	for _, c := range []struct {
		t    Target
		want string
	}{
		{Target{Browser: "vivaldi", Profile: "Default"}, "activate: Vivaldi"},
		{Target{Browser: "safari"}, "activate: Safari"},
		{Target{Browser: "app", App: "Arc"}, "activate: Arc"},
	} {
		if got := activation(c.t); got != c.want {
			t.Errorf("activation(%+v) = %q, want %q", c.t, got, c.want)
		}
	}
}
