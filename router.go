package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Decision struct {
	Rule   string
	Target Target
	Note   string
}

// decide picks the first matching rule. A rule whose Vivaldi profile cannot be resolved falls back
// instead of opening Vivaldi without a profile, which would silently use the last active one.
func decide(cfg Config, url string) Decision {
	for _, r := range cfg.Rules {
		for _, re := range r.compiled {
			if !re.MatchString(url) {
				continue
			}
			t := r.Target
			if t.Browser == "vivaldi" {
				dir, err := resolveVivaldiProfile(t)
				if err != nil {
					return Decision{Rule: "fallback", Target: cfg.Fallback, Note: fmt.Sprintf("rule %q: %v", r.Name, err)}
				}
				t.Profile = dir
			}
			return Decision{Rule: r.Name, Target: t}
		}
	}
	return Decision{Rule: "fallback", Target: cfg.Fallback}
}

func vivaldiUserDataDir(t Target) string {
	if t.UserDataDir != "" {
		return t.UserDataDir
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Vivaldi")
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Vivaldi", "User Data")
	default:
		return filepath.Join(home, ".config", "vivaldi")
	}
}

// resolveVivaldiProfile maps a profile display name ("Acme") to its directory ("Profile 1") via
// Local State; a directory name is accepted as is. Unknown names are an error, never a guess.
func resolveVivaldiProfile(t Target) (string, error) {
	udd := vivaldiUserDataDir(t)
	if t.Profile == "" {
		return "", fmt.Errorf("no Vivaldi profile configured")
	}
	if st, err := os.Stat(filepath.Join(udd, t.Profile)); err == nil && st.IsDir() {
		return t.Profile, nil
	}
	data, err := readJSON(filepath.Join(udd, "Local State"))
	if err != nil {
		return "", fmt.Errorf("Vivaldi profile %q not found: %v", t.Profile, err)
	}
	var state struct {
		Profile struct {
			InfoCache map[string]struct {
				Name string `json:"name"`
			} `json:"info_cache"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("parse Local State: %w", err)
	}
	for dir, info := range state.Profile.InfoCache {
		if info.Name == t.Profile {
			return dir, nil
		}
	}
	return "", fmt.Errorf("Vivaldi profile %q not found", t.Profile)
}
