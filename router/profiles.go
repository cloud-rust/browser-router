package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"
)

// errNotReady means the profiles could not be created now (Vivaldi runs or was never started);
// main maps it to exit code 3 so callers such as devenv can report instead of fail.
var errNotReady = errors.New("not ready")

func profilesCmd(args []string) error {
	if len(args) == 0 {
		return usageError("profiles needs list or ensure")
	}
	fs := flag.NewFlagSet("profiles "+args[0], flag.ContinueOnError)
	udd := fs.String("user-data-dir", "", "Vivaldi data dir (default: the user's)")
	dryRun := fs.Bool("dry-run", false, "only report missing profiles")
	noOpen := fs.Bool("no-open", false, "do not open new profiles once")
	if err := fs.Parse(args[1:]); err != nil {
		return usageError(err.Error())
	}
	switch args[0] {
	case "list":
		return listProfiles(*udd)
	case "ensure":
		// Arguments after -- go to Vivaldi when it opens a new profile, for tests in containers.
		names, extra := fs.Args(), []string(nil)
		for i, a := range names {
			if a == "--" {
				names, extra = names[:i], names[i+1:]
				break
			}
		}
		return ensureProfiles(*udd, names, *dryRun, *noOpen, extra)
	}
	return usageError("unknown profiles command " + args[0])
}

func listProfiles(udd string) error {
	state, err := readLocalState(vivaldiUserDataDir(udd))
	if err != nil {
		return err
	}
	profiles := state.profiles()
	dirs := make([]string, 0, len(profiles))
	for d := range profiles {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		fmt.Printf("%s\t%s\n", profiles[d], d)
	}
	return nil
}

// ensureProfiles makes sure a Vivaldi profile exists for each display name. Existing profiles are
// taken over by name; missing ones are added to Local State while Vivaldi is closed and opened
// once, because a profile's very first start shows a welcome page that would swallow a link.
func ensureProfiles(uddFlag string, names []string, dryRun, noOpen bool, vivaldiArgs []string) error {
	if len(names) == 0 {
		return usageError("profiles ensure needs at least one profile name")
	}
	udd := vivaldiUserDataDir(uddFlag)
	state, err := readLocalState(udd)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: Vivaldi has never been started with %s; start it once", errNotReady, udd)
	} else if err != nil {
		return err
	}
	var missing []string
	for _, n := range uniq(names) {
		if dir := state.dirOf(n); dir != "" {
			fmt.Printf("exists: %s (%s)\n", n, dir)
		} else {
			missing = append(missing, n)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if dryRun {
		for _, n := range missing {
			fmt.Printf("missing: %s\n", n)
		}
		return nil
	}
	if vivaldiRunning(udd) {
		return fmt.Errorf("%w: quit Vivaldi to create the profiles %q", errNotReady, missing)
	}
	var created []string
	for _, n := range missing {
		dir, err := state.addProfile(udd, n)
		if err != nil {
			return err
		}
		created = append(created, dir)
		fmt.Printf("created: %s (%s)\n", n, dir)
	}
	if err := state.save(); err != nil {
		return err
	}
	if noOpen {
		return nil
	}
	for i, dir := range created {
		if i > 0 {
			time.Sleep(3 * time.Second) // let the first start take the singleton before the next hands over
		}
		t := Target{Browser: "vivaldi", Profile: dir, UserDataDir: uddFlag, Args: vivaldiArgs}
		if err := launch(t, ""); err != nil {
			return fmt.Errorf("open %s: %w", dir, err)
		}
	}
	return nil
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
