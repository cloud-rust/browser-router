package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Target names where a URL goes. Only Vivaldi supports a profile; every other browser or app
// is opened as is, so the personal fallback can be any browser the user prefers.
type Target struct {
	Browser     string   `json:"browser"`                 // vivaldi | firefox | safari | chrome | app
	Profile     string   `json:"profile,omitempty"`       // Vivaldi profile display name or directory
	App         string   `json:"app,omitempty"`           // browser=app: app name (macOS) or executable path
	UserDataDir string   `json:"user_data_dir,omitempty"` // isolated Vivaldi data dir, for tests
	Args        []string `json:"args,omitempty"`          // extra arguments before the URL
}

type Rule struct {
	Name   string   `json:"name"`
	Match  []string `json:"match,omitempty"` // globs as in Finicky: * is the only wildcard, case-insensitive
	Regex  []string `json:"regex,omitempty"` // regular expressions against the full URL, case-insensitive
	URLs   []string `json:"urls,omitempty"`  // host and path prefix (urls.go), before all match/regex; most specific wins
	Target Target   `json:"target"`

	compiled []*regexp.Regexp
	sites    []siteURL
}

type Config struct {
	Rules []Rule `json:"rules"`
	// Clients names the target per client for links that come with a client, such as the sign-in
	// of a command line tool called in a client's folder (--client or BROWSER_ROUTER_CLIENT).
	Clients  map[string]Target `json:"clients,omitempty"`
	Fallback Target            `json:"fallback"`
}

var browsers = map[string]bool{"vivaldi": true, "firefox": true, "safari": true, "chrome": true, "app": true}

func configDir() string {
	if p := os.Getenv("BROWSER_ROUTER_CONFIG"); p != "" {
		return filepath.Dir(p)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "browser-router")
}

func configPath() string {
	if p := os.Getenv("BROWSER_ROUTER_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(configDir(), "config.json")
}

// builtinFallback is the personal browser used when the config is missing or broken.
func builtinFallback() Target {
	if runtime.GOOS == "darwin" {
		return Target{Browser: "safari"}
	}
	return Target{Browser: "firefox"}
}

func exampleConfig() Config {
	return Config{
		Rules: []Rule{
			{
				Name:   "development",
				Regex:  []string{`^https?://(localhost|127\.0\.0\.1|\[::1\]|0\.0\.0\.0|[^/?#]+\.localhost)(:\d+)?([/?#]|$)`},
				Target: Target{Browser: "firefox"},
			},
			{
				Name:   "example client",
				Match:  []string{"*example.com*"},
				Target: Target{Browser: "vivaldi", Profile: "Example"},
			},
		},
		Fallback: builtinFallback(),
	}
}

// loadConfig reads the user's config file. Without one, every link goes to the built-in fallback.
func loadConfig() (Config, bool, error) {
	data, err := readJSON(configPath())
	if errors.Is(err, os.ErrNotExist) {
		return Config{Fallback: builtinFallback()}, false, nil
	} else if err != nil {
		return Config{}, false, err
	}
	cfg, err := parseConfig(data)
	if err != nil {
		return Config{}, true, fmt.Errorf("%s: %w", configPath(), err)
	}
	return cfg, true, nil
}

func parseConfig(data []byte) (Config, error) {
	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.Fallback.Browser == "" {
		cfg.Fallback = builtinFallback()
	}
	return cfg, compile(&cfg)
}

// readJSON drops a UTF-8 byte order mark, which Windows tools such as PowerShell 5 write.
func readJSON(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	return bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), err
}

func compile(cfg *Config) error {
	if err := checkTarget(cfg.Fallback); err != nil {
		return fmt.Errorf("fallback: %w", err)
	}
	if cfg.Fallback.Browser == "vivaldi" {
		return fmt.Errorf("fallback: vivaldi needs a profile, which may be missing; use another browser")
	}
	for key, t := range cfg.Clients {
		if err := checkTarget(t); err != nil {
			return fmt.Errorf("client %q: %w", key, err)
		}
	}
	owner := map[string]string{} // an entry in two rules would make the result depend on their order
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		if err := checkTarget(r.Target); err != nil {
			return fmt.Errorf("rule %q: %w", r.Name, err)
		}
		r.sites = nil
		for _, raw := range r.URLs {
			u, err := parseSiteURL(raw)
			if err != nil {
				return fmt.Errorf("rule %q: %w", r.Name, err)
			}
			if other, ok := owner[u.String()]; ok && other != r.Name {
				return fmt.Errorf("rule %q: url %q is also in rule %q", r.Name, u, other)
			}
			owner[u.String()] = r.Name
			r.sites = append(r.sites, u)
		}
		r.compiled = nil
		for _, g := range r.Match {
			r.compiled = append(r.compiled, globRegexp(g))
		}
		for _, m := range r.Regex {
			re, err := regexp.Compile("(?i)" + m)
			if err != nil {
				return fmt.Errorf("rule %q: %w", r.Name, err)
			}
			r.compiled = append(r.compiled, re)
		}
		if len(r.compiled) == 0 && len(r.sites) == 0 {
			return fmt.Errorf("rule %q: no urls, match or regex", r.Name)
		}
	}
	return nil
}

func checkTarget(t Target) error {
	switch {
	case !browsers[t.Browser]:
		return fmt.Errorf("unknown browser %q", t.Browser)
	case t.Browser == "vivaldi" && t.Profile == "":
		return fmt.Errorf("vivaldi needs a profile: without one Vivaldi opens the last used profile")
	case t.Browser == "app" && t.App == "":
		return fmt.Errorf("browser app needs app")
	}
	return nil
}

// globRegexp turns a Finicky-style pattern into a regular expression: * matches anything, the
// pattern covers the whole URL, and a pattern without a scheme also matches with any scheme in
// front, so "localhost*" matches "http://localhost:3000/".
func globRegexp(glob string) *regexp.Regexp {
	parts := strings.Split(glob, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	prefix := ""
	if !strings.Contains(glob, "://") {
		prefix = `(?:[a-z][a-z0-9+.-]*:(?://)?)?`
	}
	return regexp.MustCompile(`(?is)^` + prefix + strings.Join(parts, ".*") + `$`)
}
