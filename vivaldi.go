package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

func vivaldiUserDataDir(override string) string {
	if override != "" {
		return override
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

// localState is Vivaldi's "Local State" file, kept as a generic tree so that writing it back
// preserves every field this program does not know about.
type localState struct {
	path string
	data map[string]any
}

func readLocalState(udd string) (*localState, error) {
	path := filepath.Join(udd, "Local State")
	raw, err := readJSON(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // keep large integers such as timestamps exact
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("parse Local State: %w", err)
	}
	return &localState{path: path, data: data}, nil
}

func (s *localState) section(name string) map[string]any {
	m, _ := s.data[name].(map[string]any)
	if m == nil {
		m = map[string]any{}
		s.data[name] = m
	}
	return m
}

func (s *localState) infoCache() map[string]any {
	profile := s.section("profile")
	cache, _ := profile["info_cache"].(map[string]any)
	if cache == nil {
		cache = map[string]any{}
		profile["info_cache"] = cache
	}
	return cache
}

// profiles returns directory -> display name.
func (s *localState) profiles() map[string]string {
	out := map[string]string{}
	for dir, v := range s.infoCache() {
		info, _ := v.(map[string]any)
		name, _ := info["name"].(string)
		out[dir] = name
	}
	return out
}

func (s *localState) dirOf(name string) string {
	dirs := []string{}
	for dir, n := range s.profiles() {
		if n == name {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs) // two profiles with one name: pick the same one every time
	if len(dirs) == 0 {
		return ""
	}
	return dirs[0]
}

// addProfile registers a new profile directory under its display name. Vivaldi creates the
// profile's own files when it opens the directory for the first time.
func (s *localState) addProfile(udd, name string) (string, error) {
	next := 1
	entries, _ := os.ReadDir(udd)
	taken := map[string]bool{}
	for _, e := range entries {
		taken[e.Name()] = true
	}
	for dir := range s.infoCache() {
		taken[dir] = true
	}
	for dir := range taken {
		if n, err := strconv.Atoi(strings.TrimPrefix(dir, "Profile ")); err == nil && strings.HasPrefix(dir, "Profile ") && n >= next {
			next = n + 1
		}
	}
	dir := fmt.Sprintf("Profile %d", next)
	if err := os.MkdirAll(filepath.Join(udd, dir), 0o700); err != nil {
		return "", err
	}
	s.infoCache()[dir] = map[string]any{
		"name":                  name,
		"is_using_default_name": false,
		"avatar_icon":           "chrome://theme/IDR_PROFILE_AVATAR_26",
	}
	profile := s.section("profile")
	if order, ok := profile["profiles_order"].([]any); ok {
		profile["profiles_order"] = append(order, dir)
	}
	return dir, nil
}

// save writes atomically, so a crash never leaves Vivaldi with a truncated Local State.
func (s *localState) save() error {
	data, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if st, err := os.Stat(s.path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp := s.path + ".browser-router"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// vivaldiRunning tells whether a Vivaldi process uses this data dir. Vivaldi rewrites Local State
// on exit, so profiles may only be added while it is closed.
func vivaldiRunning(udd string) bool {
	if runtime.GOOS == "windows" {
		lock := filepath.Join(udd, "lockfile")
		// The running browser holds the lock file open, so only a stale one can be removed.
		if err := os.Remove(lock); err != nil && !errors.Is(err, os.ErrNotExist) {
			return true
		}
		return false
	}
	target, err := os.Readlink(filepath.Join(udd, "SingletonLock"))
	if err != nil {
		return false
	}
	// The link points to "<host>-<pid>"; a lock left by a crash names a process that is gone.
	pid, err := strconv.Atoi(target[strings.LastIndex(target, "-")+1:])
	if err != nil {
		return true
	}
	return processAlive(pid)
}
