package main

import (
	"fmt"
	"os"
	"path/filepath"
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

// resolveVivaldiProfile maps a profile display name ("Acme") to its directory ("Profile 1") via
// Local State; a directory name is accepted as is. Unknown names are an error, never a guess.
func resolveVivaldiProfile(t Target) (string, error) {
	udd := vivaldiUserDataDir(t.UserDataDir)
	if st, err := os.Stat(filepath.Join(udd, t.Profile)); err == nil && st.IsDir() {
		return t.Profile, nil
	}
	state, err := readLocalState(udd)
	if err != nil {
		return "", fmt.Errorf("Vivaldi profile %q not found: %v", t.Profile, err)
	}
	if dir := state.dirOf(t.Profile); dir != "" {
		return dir, nil
	}
	return "", fmt.Errorf("Vivaldi profile %q not found", t.Profile)
}
