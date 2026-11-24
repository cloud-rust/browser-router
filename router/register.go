package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appID   = "BrowserRouter"
	progID  = "BrowserRouter.URL"
	desktop = "browser-router.desktop"
)

// register installs what the OS needs to offer the router as default browser and keeps it
// current. Only with prompt does it ask to become the default: macOS shows its confirmation
// dialog, Windows opens Settings, Linux switches directly. Without prompt it never changes the
// default, so unattended runs can call it every time.
func register(prompt bool) error {
	exe, err := selfPath()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return registerMac(prompt)
	case "linux":
		return registerLinux(exe, prompt)
	case "windows":
		return registerWindows(exe, prompt)
	}
	return fmt.Errorf("register is not supported on %s", runtime.GOOS)
}

func unregister(restore string) error {
	switch runtime.GOOS {
	case "darwin":
		return unregisterMac(restore)
	case "linux":
		home, _ := os.UserHomeDir()
		if restore != "" {
			if out, err := exec.Command("xdg-settings", "set", "default-web-browser", restore).CombinedOutput(); err != nil {
				return fmt.Errorf("xdg-settings: %v %s", err, out)
			}
		}
		err := os.Remove(filepath.Join(home, ".local", "share", "applications", desktop))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	case "windows":
		for _, k := range []string{
			`HKCU\Software\Classes\` + progID,
			`HKCU\Software\Clients\StartMenuInternet\` + appID,
		} {
			_ = exec.Command("reg", "delete", k, "/f").Run()
		}
		_ = exec.Command("reg", "delete", `HKCU\Software\RegisteredApplications`, "/v", appID, "/f").Run()
		return nil
	}
	return fmt.Errorf("unregister is not supported on %s", runtime.GOOS)
}

// currentDefault names the default browser as the OS records it: a bundle id on macOS, a desktop
// file on Linux, a ProgId on Windows.
func currentDefault() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return macDefault()
	case "linux":
		out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()
		return strings.TrimSpace(string(out)), err
	case "windows":
		out, err := exec.Command("reg", "query",
			`HKCU\Software\Microsoft\Windows\Shell\Associations\UrlAssociations\https\UserChoice`, "/v", "ProgId").Output()
		if err != nil {
			return "", nil // no user choice yet: the system default, not the router
		}
		fields := strings.Fields(string(out))
		if len(fields) == 0 {
			return "", nil
		}
		return fields[len(fields)-1], nil
	}
	return "", fmt.Errorf("not supported on %s", runtime.GOOS)
}

func ownDefaultName() string {
	return map[string]string{"darwin": bundleID, "linux": desktop, "windows": progID}[runtime.GOOS]
}

func isDefault() (bool, string, error) {
	cur, err := currentDefault()
	return err == nil && cur == ownDefaultName(), cur, err
}

func reportDefault(prompt bool) (bool, error) {
	ok, cur, err := isDefault()
	if err != nil {
		return false, err
	}
	if ok {
		fmt.Println("default browser: yes")
		return true, nil
	}
	if cur == "" {
		cur = "system default"
	}
	if !prompt {
		fmt.Printf("default browser: no (current: %s); run 'browser-router register' to switch\n", cur)
	}
	return false, nil
}

// selfPath is the real binary, not a symlink or a mise shim pointing to it.
func selfPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	return exe, nil
}

// writeIfChanged leaves an unchanged file alone; it reports whether it wrote.
func writeIfChanged(path string, data []byte, mode os.FileMode) (bool, error) {
	if cur, err := os.ReadFile(path); err == nil && bytes.Equal(cur, data) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, data, mode)
}

func registerLinux(exe string, prompt bool) error {
	home, _ := os.UserHomeDir()
	// xdg-settings rejects a quoted Exec path, so quote only when the path needs it.
	if strings.ContainsAny(exe, " \t") {
		exe = `"` + exe + `"`
	}
	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Browser Router
Comment=Routes links to browsers and browser profiles
Exec=%s %%u
MimeType=x-scheme-handler/http;x-scheme-handler/https;text/html;
NoDisplay=true
`, exe)
	if _, err := writeIfChanged(filepath.Join(home, ".local", "share", "applications", desktop), []byte(entry), 0o644); err != nil {
		return err
	}
	if ok, err := reportDefault(prompt); err != nil || ok || !prompt {
		return err
	}
	for _, args := range [][]string{
		{"xdg-mime", "default", desktop, "x-scheme-handler/http"},
		{"xdg-mime", "default", desktop, "x-scheme-handler/https"},
		{"xdg-settings", "set", "default-web-browser", desktop},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %v %s", args, err, out)
		}
	}
	fmt.Println("default browser: yes")
	return nil
}

func registerWindows(exe string, prompt bool) error {
	open := fmt.Sprintf(`"%s" "%%1"`, exe)
	client := `HKCU\Software\Clients\StartMenuInternet\` + appID
	entries := [][]string{
		{`HKCU\Software\Classes\` + progID, "/ve", "/d", "Browser Router URL"},
		{`HKCU\Software\Classes\` + progID, "/v", "URL Protocol", "/d", ""},
		{`HKCU\Software\Classes\` + progID + `\shell\open\command`, "/ve", "/d", open},
		{client, "/ve", "/d", "Browser Router"},
		{client + `\Capabilities`, "/v", "ApplicationName", "/d", "Browser Router"},
		{client + `\Capabilities`, "/v", "ApplicationDescription", "/d", "Routes links to browsers and browser profiles"},
		{client + `\Capabilities\URLAssociations`, "/v", "http", "/d", progID},
		{client + `\Capabilities\URLAssociations`, "/v", "https", "/d", progID},
		{client + `\shell\open\command`, "/ve", "/d", fmt.Sprintf(`"%s"`, exe)},
		{`HKCU\Software\RegisteredApplications`, "/v", appID, "/d", `Software\Clients\StartMenuInternet\` + appID + `\Capabilities`},
	}
	for _, e := range entries {
		args := append([]string{"add", e[0]}, e[1:]...)
		args = append(args, "/f")
		if out, err := exec.Command("reg", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("reg %v: %v %s", args, err, out)
		}
	}
	if ok, err := reportDefault(prompt); err != nil || ok || !prompt {
		return err
	}
	// Windows forbids setting the default programmatically; open the page where the user picks it.
	fmt.Println("pick Browser Router as web browser in Settings")
	return exec.Command("cmd", "/c", "start", "", "ms-settings:defaultapps?registeredAppUser="+appID).Run()
}
