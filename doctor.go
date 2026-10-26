package main

import (
	"fmt"
	"runtime"
	"sort"
)

func status() error {
	fmt.Printf("browser-router %s\n", version)
	cfg, found, err := loadConfig()
	switch {
	case err != nil:
		fmt.Printf("config: %v\n", err)
	case !found:
		fmt.Printf("config: %s missing; every link goes to %s\n", configPath(), cfg.Fallback.Browser)
	default:
		fmt.Printf("config: %s (%d rules, fallback %s)\n", configPath(), len(cfg.Rules), cfg.Fallback.Browser)
	}
	if runtime.GOOS == "darwin" {
		fmt.Printf("app: %s (%s)\n", installedApp(), appState())
	}
	ok, cur, err := isDefault()
	switch {
	case err != nil:
		fmt.Printf("default browser: unknown (%v)\n", err)
	case ok:
		fmt.Println("default browser: yes")
	default:
		fmt.Printf("default browser: no (current: %s)\n", cur)
	}
	fmt.Printf("log: %s\n", logPath())
	return nil
}

// doctor checks everything a link depends on and fails when anything would send links astray.
func doctor() error {
	problems := 0
	report := func(ok bool, format string, a ...any) {
		mark := "ok     "
		if !ok {
			mark = "problem"
			problems++
		}
		fmt.Printf("%s  %s\n", mark, fmt.Sprintf(format, a...))
	}

	cfg, found, err := loadConfig()
	switch {
	case err != nil:
		report(false, "config: %v", err)
	case !found:
		report(false, "config: %s missing", configPath())
	default:
		report(true, "config: %s (%d rules)", configPath(), len(cfg.Rules))
	}

	targets := []Target{cfg.Fallback}
	for _, r := range cfg.Rules {
		targets = append(targets, r.Target)
	}
	seen := map[string]bool{}
	for _, t := range targets {
		key := t.Browser + "|" + t.App
		if t.Browser == "" || seen[key] {
			continue
		}
		seen[key] = true
		name := t.Browser
		if t.Browser == "app" {
			name = t.App
		}
		if err := available(t); err != nil {
			report(false, "browser %s: %v", name, err)
		} else {
			report(true, "browser %s", name)
		}
	}
	for _, r := range cfg.Rules {
		if r.Target.Browser != "vivaldi" {
			continue
		}
		if dir, err := resolveVivaldiProfile(r.Target); err != nil {
			report(false, "rule %q: %v (browser-router profiles ensure %q)", r.Name, err, r.Target.Profile)
		} else {
			report(true, "rule %q: Vivaldi profile %s (%s)", r.Name, r.Target.Profile, dir)
		}
	}

	for _, key := range sortedKeys(cfg.Clients) {
		t := cfg.Clients[key]
		if t.Browser != "vivaldi" {
			continue
		}
		if dir, err := resolveVivaldiProfile(t); err != nil {
			report(false, "client %q: %v (browser-router profiles ensure %q)", key, err, t.Profile)
		} else {
			report(true, "client %q: Vivaldi profile %s (%s)", key, t.Profile, dir)
		}
	}

	if runtime.GOOS == "darwin" {
		state := appState()
		report(state == "current", "app %s: %s", installedApp(), state)
	}
	ok, cur, err := isDefault()
	switch {
	case err != nil:
		report(false, "default browser: %v", err)
	case ok:
		report(true, "default browser")
	default:
		report(false, "default browser is %s (browser-router register)", cur)
	}
	if problems > 0 {
		return fmt.Errorf("%d problem(s)", problems)
	}
	return nil
}

func sortedKeys(m map[string]Target) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
