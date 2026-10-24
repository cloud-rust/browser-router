package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func command(t Target, url string) (*exec.Cmd, error) {
	var args []string
	if t.Browser == "vivaldi" {
		if t.UserDataDir != "" {
			args = append(args, "--user-data-dir="+t.UserDataDir)
		}
		args = append(args, "--profile-directory="+t.Profile)
	}
	args = append(append(args, t.Args...), url)

	switch runtime.GOOS {
	case "darwin":
		switch t.Browser {
		case "vivaldi":
			// -n is required: for a running app, open drops --args and the URL with them.
			return exec.Command("open", append([]string{"-n", "-a", "Vivaldi", "--args"}, args...)...), nil
		case "firefox", "safari":
			return exec.Command("open", append([]string{"-a", appName(t.Browser)}, args...)...), nil
		case "app":
			return exec.Command("open", append([]string{"-a", t.App}, args...)...), nil
		}
	case "windows":
		exe := t.App
		switch t.Browser {
		case "vivaldi":
			exe = firstExisting(
				filepath.Join(os.Getenv("LOCALAPPDATA"), `Vivaldi\Application\vivaldi.exe`),
				filepath.Join(os.Getenv("ProgramFiles"), `Vivaldi\Application\vivaldi.exe`))
		case "firefox":
			exe = firstExisting(filepath.Join(os.Getenv("ProgramFiles"), `Mozilla Firefox\firefox.exe`))
		}
		if exe != "" {
			return exec.Command(exe, args...), nil
		}
	default:
		exe := map[string]string{"vivaldi": "vivaldi", "firefox": "firefox", "app": t.App}[t.Browser]
		if exe != "" {
			return exec.Command(exe, args...), nil
		}
	}
	return nil, fmt.Errorf("browser %q is not available on %s", t.Browser, runtime.GOOS)
}

func appName(browser string) string {
	return map[string]string{"firefox": "Firefox", "safari": "Safari"}[browser]
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// launch starts the browser and returns without waiting, so the router never blocks the caller.
func launch(t Target, url string) error {
	cmd, err := command(t, url)
	if err != nil {
		return err
	}
	return cmd.Start()
}
