package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

// Target names where a URL goes. Only Vivaldi supports a profile; every other browser or app
// is opened as is, so the personal fallback can be any browser the user prefers.
type Target struct {
	Browser     string   `json:"browser"`                 // vivaldi | firefox | safari | app
	Profile     string   `json:"profile,omitempty"`       // Vivaldi profile display name or directory
	App         string   `json:"app,omitempty"`           // browser=app: app name (macOS) or executable path
	UserDataDir string   `json:"user_data_dir,omitempty"` // isolated Vivaldi data dir, for tests
	Args        []string `json:"args,omitempty"`          // extra arguments before the URL
}

type Rule struct {
	Name   string   `json:"name"`
	Match  []string `json:"match"` // regular expressions against the full URL, case-insensitive
	Target Target   `json:"target"`

	compiled []*regexp.Regexp
}

type Config struct {
	Rules    []Rule `json:"rules"`
	Fallback Target `json:"fallback"`
}

func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "browser-router-poc")
}

func configPath() string { return filepath.Join(configDir(), "config.json") }

func defaultConfig() Config {
	fallback := Target{Browser: "firefox"}
	if runtime.GOOS == "darwin" {
		fallback = Target{Browser: "safari"}
	}
	return Config{
		Rules: []Rule{
			{
				Name:   "development",
				Match:  []string{`^https?://(localhost|127\.0\.0\.1|\[::1\]|[^/]+\.localhost)(:\d+)?([/?#]|$)`},
				Target: Target{Browser: "firefox"},
			},
			{
				Name:   "acme",
				Match:  []string{`acme`},
				Target: Target{Browser: "vivaldi", Profile: "Acme"},
			},
		},
		Fallback: fallback,
	}
}

// loadConfig reads the user's config file, falling back to the built-in example when none exists.
func loadConfig() (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err == nil {
		cfg = Config{}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", configPath(), err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	return cfg, compile(&cfg)
}

func compile(cfg *Config) error {
	for i := range cfg.Rules {
		cfg.Rules[i].compiled = nil
		for _, m := range cfg.Rules[i].Match {
			re, err := regexp.Compile("(?i)" + m)
			if err != nil {
				return fmt.Errorf("rule %q: %w", cfg.Rules[i].Name, err)
			}
			cfg.Rules[i].compiled = append(cfg.Rules[i].compiled, re)
		}
	}
	return nil
}
