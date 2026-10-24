package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appID   = "BrowserRouterPoC"
	progID  = "BrowserRouterPoC.URL"
	desktop = "browser-router-poc.desktop"
)

// register makes the router a candidate default browser. Linux switches immediately; Windows only
// lets the user pick it in Settings; macOS is handled by the app bundle (mac/), not this binary.
func register() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return registerLinux(exe)
	case "windows":
		return registerWindows(exe)
	default:
		return fmt.Errorf("on macOS run: open BrowserRouterPoC.app --args --register")
	}
}

func unregister() error {
	switch runtime.GOOS {
	case "linux":
		home, _ := os.UserHomeDir()
		return os.Remove(filepath.Join(home, ".local", "share", "applications", desktop))
	case "windows":
		for _, k := range []string{
			`HKCU\Software\Classes\` + progID,
			`HKCU\Software\Clients\StartMenuInternet\` + appID,
		} {
			_ = exec.Command("reg", "delete", k, "/f").Run()
		}
		return exec.Command("reg", "delete", `HKCU\Software\RegisteredApplications`, "/v", appID, "/f").Run()
	default:
		return fmt.Errorf("on macOS run: open BrowserRouterPoC.app --args --restore <bundle id>")
	}
}

func registerLinux(exe string) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// xdg-settings rejects a quoted Exec path, so quote only when the path needs it.
	if strings.ContainsAny(exe, " \t") {
		exe = `"` + exe + `"`
	}
	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Browser Router PoC
Exec=%s %%u
MimeType=x-scheme-handler/http;x-scheme-handler/https;
NoDisplay=true
`, exe)
	if err := os.WriteFile(filepath.Join(dir, desktop), []byte(entry), 0o644); err != nil {
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
	return nil
}

func registerWindows(exe string) error {
	open := fmt.Sprintf(`"%s" "%%1"`, exe)
	client := `HKCU\Software\Clients\StartMenuInternet\` + appID
	entries := [][]string{
		{`HKCU\Software\Classes\` + progID, "/ve", "/d", "Browser Router PoC URL"},
		{`HKCU\Software\Classes\` + progID, "/v", "URL Protocol", "/d", ""},
		{`HKCU\Software\Classes\` + progID + `\shell\open\command`, "/ve", "/d", open},
		{client, "/ve", "/d", "Browser Router PoC"},
		{client + `\Capabilities`, "/v", "ApplicationName", "/d", "Browser Router PoC"},
		{client + `\Capabilities`, "/v", "ApplicationDescription", "/d", "Routes links to browsers and profiles"},
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
	// Windows forbids setting the default programmatically; open the page where the user picks it.
	return exec.Command("cmd", "/c", "start", "", "ms-settings:defaultapps?registeredAppUser="+appID).Run()
}
