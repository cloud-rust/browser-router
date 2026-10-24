package main

import (
	"os"
	"path/filepath"
	"testing"
)

func testConfig(t *testing.T, udd string) Config {
	t.Helper()
	cfg := defaultConfig()
	cfg.Rules[1].Target.UserDataDir = udd
	cfg.Fallback = Target{Browser: "safari"}
	if err := compile(&cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestDecide(t *testing.T) {
	udd := t.TempDir()
	// With a byte order mark, as PowerShell 5 writes it.
	if err := os.WriteFile(filepath.Join(udd, "Local State"),
		[]byte("\xef\xbb\xbf"+`{"profile":{"info_cache":{"Profile 1":{"name":"Acme"}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(t, udd)

	cases := []struct{ url, rule, browser, profile string }{
		{"http://localhost:3000/x", "development", "firefox", ""},
		{"http://127.0.0.1:8080", "development", "firefox", ""},
		{"http://[::1]:5000/", "development", "firefox", ""},
		{"http://app.localhost:3000/", "development", "firefox", ""},
		{"https://localhost.evil.com/", "fallback", "safari", ""},
		{"https://portal.azure.com/#@ACME.onmicrosoft.com", "acme", "vivaldi", "Profile 1"},
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
