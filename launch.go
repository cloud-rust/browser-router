package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var macApps = map[string]string{"vivaldi": "Vivaldi", "firefox": "Firefox", "safari": "Safari", "chrome": "Google Chrome"}

// command builds the call that opens url in t. An empty url just opens the browser, which is how
// a new Vivaldi profile gets its first start.
func command(t Target, url string) (*exec.Cmd, error) {
	var args []string
	if t.Browser == "vivaldi" {
		if t.UserDataDir != "" {
			args = append(args, "--user-data-dir="+t.UserDataDir)
		}
		args = append(args, "--profile-directory="+t.Profile)
	}
	args = append(args, t.Args...)
	if url != "" {
		args = append(args, url)
	}

	switch runtime.GOOS {
	case "darwin":
		app := macApps[t.Browser]
		if t.Browser == "app" {
			app = t.App
		}
		if app == "" {
			break
		}
		if t.Browser == "vivaldi" {
			// -n is required: for a running app, open drops --args and the URL with them.
			return exec.Command("open", append([]string{"-n", "-a", app, "--args"}, args...)...), nil
		}
		return exec.Command("open", append([]string{"-a", app}, args...)...), nil
	case "windows":
		if exe := executable(t); exe != "" {
			return exec.Command(exe, args...), nil
		}
	default:
		if exe := executable(t); exe != "" {
			return exec.Command(exe, args...), nil
		}
	}
	return nil, fmt.Errorf("browser %q is not available on %s", t.Browser, runtime.GOOS)
}

// executable finds the program for t on Linux and Windows; empty when it is not installed.
func executable(t Target) string {
	if runtime.GOOS == "windows" {
		local, programs := os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramFiles")
		switch t.Browser {
		case "vivaldi":
			return firstExisting(filepath.Join(local, `Vivaldi\Application\vivaldi.exe`),
				filepath.Join(programs, `Vivaldi\Application\vivaldi.exe`))
		case "firefox":
			return firstExisting(filepath.Join(programs, `Mozilla Firefox\firefox.exe`))
		case "chrome":
			return firstExisting(filepath.Join(programs, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(local, `Google\Chrome\Application\chrome.exe`))
		case "app":
			return lookPath(t.App)
		}
		return ""
	}
	names := map[string][]string{
		"vivaldi": {"vivaldi", "vivaldi-stable"},
		"firefox": {"firefox"},
		"chrome":  {"google-chrome", "google-chrome-stable", "chromium"},
		"app":     {t.App},
	}[t.Browser]
	for _, n := range names {
		if p := lookPath(n); p != "" {
			return p
		}
	}
	return ""
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// available reports whether the browser or app of t is installed, without opening it.
func available(t Target) error {
	if runtime.GOOS == "darwin" {
		app := macApps[t.Browser]
		if t.Browser == "app" {
			app = t.App
		}
		if app == "" || exec.Command("open", "-Ra", app).Run() != nil {
			return fmt.Errorf("app %q not found", app)
		}
		return nil
	}
	if executable(t) == "" {
		return fmt.Errorf("browser %q not found", t.Browser)
	}
	return nil
}

// launch starts the browser and returns without waiting, so the router never blocks the caller.
func launch(t Target, url string) error {
	cmd, err := command(t, url)
	if err != nil {
		return err
	}
	return cmd.Start()
}
