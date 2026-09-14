package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// npmManagedPath — heuristic: does this path live somewhere npm owns
// (global node_modules, the @yyx462 scope dir, npx cache)? The wrapper
// resolves its bin into those; a curl/brew/custom install does not.
func npmManagedPath(p string) bool {
	p = filepath.ToSlash(p)
	return strings.Contains(p, "node_modules/") ||
		strings.Contains(p, "@yyx462/") ||
		strings.Contains(p, "lib/node_modules") ||
		strings.Contains(p, "/npm-global/")
}

func npmDetected() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return npmManagedPath(exe)
}

// upgradeCommand — the installer line matching how this binary got
// here. npm installs stay npm (the wrapper owns the bin); everything
// else rides the curl/PowerShell one-liner from the README.
func upgradeCommand() (desc string, argv []string) {
	switch {
	case npmDetected():
		return "npm i -g @yyx462/log-labor@latest", []string{"npm", "i", "-g", "@yyx462/log-labor@latest"}
	case runtime.GOOS == "windows":
		s := "irm https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.ps1 | iex"
		return s, []string{"powershell", "-NoProfile", "-Command", s}
	default:
		s := "curl -fsSL https://raw.githubusercontent.com/yyx462/AutoWecom-plugin/master/log-labor/install.sh | sh"
		return s, []string{"sh", "-c", s}
	}
}

// cmdUpgrade — explicit self-update: rerun the installer that put this
// binary here. Industry thumb: the verb is consent — never a silent
// background swap (--yes skips the confirm, --dry-run prints the plan).
// Agent skills re-render themselves on the first command after the
// upgrade (autoSkillRefresh), so upgrade is genuinely one step.
func cmdUpgrade(args []string) error {
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	desc, argv := upgradeCommand()
	fmt.Printf("current:  log-labor %s\n", version)
	fmt.Printf("upgrade:  %s\n", desc)
	if f.has("dry-run") {
		fmt.Println("dry-run — nothing run.")
		return nil
	}
	ok, err := confirm("run the upgrade?", f.has("yes"))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("aborted — nothing run.")
		return nil
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return exitError{code: 1, msg: fmt.Sprintf("upgrade failed: %v", err)}
	}
	fmt.Println("upgraded — confirm with `log-labor version`; agent skills refresh on the next command.")
	return nil
}
