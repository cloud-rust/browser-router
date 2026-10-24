// Command browser-router is a cross-platform browser router: registered as the default browser,
// it sends each link to a browser or Vivaldi profile according to rules.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var version = "dev" // set by the release build

const usage = `usage:
  browser-router <url>                  route a link (what the OS calls as default browser)
  browser-router test <url>             print where a link would go, without opening it
  browser-router register [--no-prompt] install for the OS and ask to become the default browser;
                                        --no-prompt only installs and reports, it never asks
  browser-router unregister [<browser>] remove the registration (macOS: restore <bundle id>, default Safari)
  browser-router status                 version, config, default browser
  browser-router doctor                 check config, browsers, Vivaldi profiles and registration
  browser-router profiles list          Vivaldi profiles: name and directory
  browser-router profiles ensure [--dry-run] [--no-open] [--user-data-dir DIR] <name>... [-- <vivaldi args>]
                                        create missing Vivaldi profiles (Vivaldi must be closed)
  browser-router config path|init|check config file location, write the example, validate
  browser-router version

exit codes: 0 ok, 1 error, 2 usage, 3 not possible now (profiles ensure: Vivaldi runs)`

type usageError string

func (e usageError) Error() string { return string(e) }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	// Only routing logs: as default browser the router has no terminal; the commands print instead
	var logger *log.Logger
	args := os.Args[2:]

	var err error
	switch cmd := os.Args[1]; {
	case cmd == "test" && len(args) == 1:
		err = test(args[0])
	case cmd == "register" && (len(args) == 0 || len(args) == 1 && args[0] == "--no-prompt"):
		err = register(len(args) == 0)
	case cmd == "unregister" && len(args) <= 1:
		err = unregister(strings.Join(args, ""))
	case cmd == "status" && len(args) == 0:
		err = status()
	case cmd == "doctor" && len(args) == 0:
		err = doctor()
	case cmd == "profiles":
		err = profilesCmd(args)
	case cmd == "config" && len(args) == 1:
		err = configCmd(args[0])
	case cmd == "version" || cmd == "--version":
		fmt.Println(version)
	case cmd == "-h" || cmd == "--help" || cmd == "help":
		fmt.Println(usage)
	case strings.Contains(cmd, ":") && len(args) == 0:
		logger = openLog()
		err = route(logger, cmd)
	default:
		err = usageError("unknown command")
	}
	var ue usageError
	switch {
	case err == nil:
	case errors.As(err, &ue):
		fmt.Fprintf(os.Stderr, "error: %v\n\n%s\n", err, usage)
		os.Exit(2)
	case errors.Is(err, errNotReady):
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	default:
		if logger != nil {
			logger.Printf("error: %v", err)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func route(logger *log.Logger, link string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		// A broken config must not swallow the link: open it in the built-in personal browser.
		logger.Printf("error: %v; using built-in fallback", err)
		return launch(builtinFallback(), link)
	}
	d := decide(cfg, link)
	if d.Note != "" {
		logger.Printf("warning: %s", d.Note)
	}
	// Only the host: full links may carry tokens, and the log stays on disk.
	logger.Printf("%s -> %s %s: %s", d.Rule, d.Target.Browser, d.Target.Profile, hostOf(link))
	if err := launch(d.Target, link); err != nil {
		if d.Rule == "fallback" {
			return err
		}
		// A missing browser must not swallow the link.
		logger.Printf("warning: %v; using fallback", err)
		return launch(cfg.Fallback, link)
	}
	return nil
}

func hostOf(link string) string {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return strings.SplitN(link, ":", 2)[0] + ":"
	}
	return u.Scheme + "://" + u.Host
}

func test(link string) error {
	cfg, found, err := loadConfig()
	if err != nil {
		return err
	}
	if !found {
		fmt.Printf("note: no config at %s\n", configPath())
	}
	d := decide(cfg, link)
	fmt.Printf("rule: %s\nbrowser: %s\nprofile: %s\n", d.Rule, d.Target.Browser, d.Target.Profile)
	if d.Note != "" {
		fmt.Printf("note: %s\n", d.Note)
	}
	cmd, err := command(d.Target, link)
	if err != nil {
		return err
	}
	fmt.Printf("command: %q\n", cmd.Args)
	return nil
}

func configCmd(sub string) error {
	switch sub {
	case "path":
		fmt.Println(configPath())
		return nil
	case "init":
		if _, err := os.Stat(configPath()); err == nil {
			return fmt.Errorf("%s already exists", configPath())
		}
		if err := os.MkdirAll(configDir(), 0o755); err != nil {
			return err
		}
		data, _ := json.MarshalIndent(exampleConfig(), "", "  ")
		if err := os.WriteFile(configPath(), append(data, '\n'), 0o644); err != nil {
			return err
		}
		fmt.Println(configPath())
		return nil
	case "check":
		cfg, found, err := loadConfig()
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("no config at %s", configPath())
		}
		fmt.Printf("%s: %d rules, fallback %s\n", configPath(), len(cfg.Rules), cfg.Fallback.Browser)
		return nil
	}
	return usageError("unknown config command " + sub)
}

func logPath() string { return filepath.Join(configDir(), "router.log") }

// openLog appends to router.log next to the config: as default browser the router has no
// terminal. The log starts over above 1 MB, keeping one previous file.
func openLog() *log.Logger {
	_ = os.MkdirAll(configDir(), 0o755)
	if st, err := os.Stat(logPath()); err == nil && st.Size() > 1<<20 {
		_ = os.Rename(logPath(), logPath()+".1")
	}
	f, err := os.OpenFile(logPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return log.New(os.Stderr, "", log.LstdFlags)
	}
	return log.New(f, "", log.LstdFlags)
}
