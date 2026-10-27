package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Decision struct {
	Rule   string
	Target Target
	Note   string
}

// decide sends a link to the rule with the most specific urls entry, else to the first rule whose
// match or regex fits. A rule whose Vivaldi profile cannot be resolved falls back instead of
// opening Vivaldi without a profile, which would silently use the last active one.
func decide(cfg Config, url string) Decision {
	if i := bestURLRule(cfg, url); i >= 0 {
		return use(cfg, cfg.Rules[i])
	}
	for _, r := range cfg.Rules {
		for _, re := range r.compiled {
			if re.MatchString(url) {
				return use(cfg, r)
			}
		}
	}
	return Decision{Rule: "fallback", Target: cfg.Fallback}
}

func use(cfg Config, r Rule) Decision {
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

// decideFor sends a link that comes with a client to that client's target. An unknown client or
// a profile that cannot be resolved leaves the link to the rules, so nothing is guessed.
func decideFor(cfg Config, client, url string) Decision {
	t, ok := cfg.Clients[client]
	if client == "" || !ok {
		d := decide(cfg, url)
		if client != "" {
			d.Note = strings.TrimPrefix(d.Note+"; ", "; ") + fmt.Sprintf("unknown client %q", client)
		}
		return d
	}
	if t.Browser == "vivaldi" {
		dir, err := resolveVivaldiProfile(t)
		if err != nil {
			d := decide(cfg, url)
			d.Note = fmt.Sprintf("client %q: %v", client, err)
			return d
		}
		t.Profile = dir
	}
	return Decision{Rule: "client " + client, Target: t}
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
