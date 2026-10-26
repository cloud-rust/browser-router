package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// On macOS, LaunchServices hands links to an app bundle as Apple Events, so the router comes with
// a small app (mac/) that passes each link to its own copy of this binary. The release archive
// carries the app next to the binary; register copies it to ~/Applications, where it stays put
// when mise installs a new version elsewhere.
const (
	bundleID   = "io.github.cloud-rust.browser-router"
	appBundle  = "Browser Router.app"
	appShell   = "BrowserRouter"
	lsregister = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
)

func installedApp() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Applications", appBundle)
}

// bundledApp is the app shipped with this binary: the app around it, or the one next to it.
func bundledApp() string {
	exe, err := selfPath()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "MacOS" {
		if app := filepath.Dir(filepath.Dir(dir)); strings.HasSuffix(app, ".app") {
			return app
		}
	}
	if st, err := os.Stat(filepath.Join(dir, appBundle)); err == nil && st.IsDir() {
		return filepath.Join(dir, appBundle)
	}
	return ""
}

// appState compares the installed app with the bundled one: current, outdated or missing.
func appState() string {
	dst := installedApp()
	dstHash, err := treeHash(dst)
	if err != nil {
		return "missing"
	}
	src := bundledApp()
	if src == "" || src == dst {
		return "current"
	}
	if srcHash, err := treeHash(src); err == nil && srcHash != dstHash {
		return "outdated"
	}
	return "current"
}

func registerMac(prompt bool) error {
	src, dst := bundledApp(), installedApp()
	switch state := appState(); {
	case src == "" && state == "missing":
		return fmt.Errorf("%s not found next to the binary; install a release archive", appBundle)
	case state != "current":
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		// ditto keeps the code signature and extended attributes intact.
		if out, err := exec.Command("ditto", src, dst).CombinedOutput(); err != nil {
			return fmt.Errorf("ditto: %v %s", err, out)
		}
		fmt.Printf("installed %s\n", dst)
	}
	if out, err := exec.Command(lsregister, "-f", dst).CombinedOutput(); err != nil {
		return fmt.Errorf("lsregister: %v %s", err, out)
	}
	if ok, err := reportDefault(prompt); err != nil || ok || !prompt {
		return err
	}
	// macOS asks the user to confirm a new default browser; the app triggers that dialog.
	fmt.Println("confirm Browser Router as default browser in the macOS dialog")
	return exec.Command("open", "-n", "-a", dst, "--args", "--register").Run()
}

func unregisterMac(restore string) error {
	if restore == "" {
		restore = "com.apple.Safari"
	}
	fmt.Printf("confirm %s as default browser in the macOS dialog\n", restore)
	return exec.Command("open", "-n", "-a", installedApp(), "--args", "--restore", restore).Run()
}

// macDefault asks the installed app, or before installing the bundled one, which app opens http
// and https links; the answer counts only when both agree.
func macDefault() (string, error) {
	app := installedApp()
	if _, err := os.Stat(app); err != nil && bundledApp() != "" {
		app = bundledApp()
	}
	out, err := exec.Command(filepath.Join(app, "Contents", "MacOS", appShell), "--print-default").Output()
	if err != nil {
		return "", fmt.Errorf("query default browser via %s: %v", appBundle, err)
	}
	ids := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f := strings.Fields(line); len(f) == 2 {
			ids[f[0]] = f[1]
		}
	}
	if ids["http"] != ids["https"] {
		return fmt.Sprintf("http: %s, https: %s", ids["http"], ids["https"]), nil
	}
	return ids["https"], nil
}

// treeHash covers names, modes and contents of every file below dir.
func treeHash(dir string) (string, error) {
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%o\x00", rel, info.Mode())
		if d.Type().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(h, f); err != nil {
				return err
			}
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}
