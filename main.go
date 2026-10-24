// Command router is a proof of concept for a cross-platform browser router: registered as the
// default browser, it sends each link to a browser or Vivaldi profile according to rules.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const usage = `usage:
  router <url>            route a link (what the OS calls as default browser)
  router test <url>       print where a link would go, without opening it
  router register         offer the router as default browser (Linux, Windows)
  router unregister       remove the registration (Linux, Windows)
  router init-config      write the example config for editing`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	logger := openLog()

	var err error
	switch cmd := os.Args[1]; {
	case cmd == "test" && len(os.Args) == 3:
		err = test(os.Args[2])
	case cmd == "register":
		err = register()
	case cmd == "unregister":
		err = unregister()
	case cmd == "init-config":
		err = initConfig()
	case strings.Contains(cmd, ":"):
		err = route(logger, cmd)
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		logger.Printf("error: %v", err)
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func route(logger *log.Logger, url string) error {
	cfg, err := loadConfig()
	if err != nil {
		// A broken config must not swallow the link: open it in the built-in personal browser.
		logger.Printf("error: %v; using built-in fallback", err)
		return launch(defaultConfig().Fallback, url)
	}
	d := decide(cfg, url)
	if d.Note != "" {
		logger.Printf("warning: %s", d.Note)
	}
	logger.Printf("%s -> %s %s: %s", d.Rule, d.Target.Browser, d.Target.Profile, url)
	if err := launch(d.Target, url); err != nil && d.Rule != "fallback" {
		// A missing browser must not swallow the link.
		logger.Printf("warning: %v; using fallback", err)
		return launch(cfg.Fallback, url)
	} else if err != nil {
		return err
	}
	return nil
}

func test(url string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	d := decide(cfg, url)
	fmt.Printf("rule: %s\nbrowser: %s\nprofile: %s\n", d.Rule, d.Target.Browser, d.Target.Profile)
	if d.Note != "" {
		fmt.Printf("note: %s\n", d.Note)
	}
	cmd, err := command(d.Target, url)
	if err != nil {
		return err
	}
	fmt.Printf("command: %q\n", cmd.Args)
	return nil
}

func initConfig() error {
	if _, err := os.Stat(configPath()); err == nil {
		return fmt.Errorf("%s already exists", configPath())
	}
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(defaultConfig(), "", "  ")
	if err := os.WriteFile(configPath(), append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println(configPath())
	return nil
}

// openLog appends to router.log next to the config: as default browser the router has no terminal.
func openLog() *log.Logger {
	_ = os.MkdirAll(configDir(), 0o755)
	f, err := os.OpenFile(filepath.Join(configDir(), "router.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return log.New(os.Stderr, "", log.LstdFlags)
	}
	return log.New(f, "", log.LstdFlags)
}
