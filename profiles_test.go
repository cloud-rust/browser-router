package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const sampleState = `{"browser":{"last_known_version":"7.0"},"profile":{"info_cache":{"Default":{"name":"Persönlich","active_time":1729771234.123456},"Profile 1":{"name":"Acme"}},"profiles_order":["Default","Profile 1"],"last_used":"Default"},"big":12345678901234567890}`

func TestEnsureProfiles(t *testing.T) {
	udd := t.TempDir()
	writeLocalState(t, udd, sampleState)
	if err := os.Mkdir(filepath.Join(udd, "Profile 4"), 0o700); err != nil { // stray dir: numbering skips it
		t.Fatal(err)
	}
	if err := ensureProfiles(udd, []string{"Acme", "Globex", "Globex Admin", "Globex"}, false, true, nil); err != nil {
		t.Fatal(err)
	}
	state, err := readLocalState(udd)
	if err != nil {
		t.Fatal(err)
	}
	got := state.profiles()
	want := map[string]string{"Default": "Persönlich", "Profile 1": "Acme", "Profile 5": "Globex", "Profile 6": "Globex Admin"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for dir, name := range want {
		if got[dir] != name {
			t.Errorf("%s: got %q, want %q", dir, got[dir], name)
		}
	}
	for _, dir := range []string{"Profile 5", "Profile 6"} {
		if _, err := os.Stat(filepath.Join(udd, dir)); err != nil {
			t.Error(err)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(udd, "Local State"))
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "12345678901234567890") || !strings.Contains(string(raw), "1729771234.123456") {
		t.Errorf("numbers changed: %s", raw)
	}
	order := data["profile"].(map[string]any)["profiles_order"].([]any)
	if len(order) != 4 || order[3] != "Profile 6" {
		t.Errorf("profiles_order: %v", order)
	}

	// A second run changes nothing.
	before, _ := os.ReadFile(filepath.Join(udd, "Local State"))
	if err := ensureProfiles(udd, []string{"Acme", "Globex"}, false, true, nil); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(udd, "Local State"))
	if string(before) != string(after) {
		t.Error("second run rewrote Local State")
	}
}

func TestEnsureProfilesDryRunAndNotReady(t *testing.T) {
	udd := t.TempDir()
	if err := ensureProfiles(udd, []string{"Acme"}, false, true, nil); !errors.Is(err, errNotReady) {
		t.Errorf("no Local State: got %v, want errNotReady", err)
	}
	writeLocalState(t, udd, sampleState)
	if err := ensureProfiles(udd, []string{"New"}, true, true, nil); err != nil {
		t.Fatal(err)
	}
	if state, _ := readLocalState(udd); state.dirOf("New") != "" {
		t.Error("dry run created a profile")
	}
	if runtime.GOOS == "windows" {
		return
	}
	// A lock of a live process (this test) means Vivaldi runs.
	if err := os.Symlink("host-"+strconv.Itoa(os.Getpid()), filepath.Join(udd, "SingletonLock")); err != nil {
		t.Fatal(err)
	}
	if err := ensureProfiles(udd, []string{"New"}, false, true, nil); !errors.Is(err, errNotReady) {
		t.Errorf("running Vivaldi: got %v, want errNotReady", err)
	}
	// A lock left by a crash does not block.
	_ = os.Remove(filepath.Join(udd, "SingletonLock"))
	if err := os.Symlink("host-999999999", filepath.Join(udd, "SingletonLock")); err != nil {
		t.Fatal(err)
	}
	if err := ensureProfiles(udd, []string{"New"}, false, true, nil); err != nil {
		t.Errorf("stale lock: %v", err)
	}
}
