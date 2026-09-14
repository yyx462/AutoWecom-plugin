// log-labor — 报工 CLI for the team labor smartsheet (WeCom 智能表格
// webhook). Pure stdlib; the webhook contract is live-proved — see
// internal/webhook and skill/SKILL.md.tmpl.
package main

import (
	"fmt"
	"os"
	"strings"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/skillinstall"
)

var version = config.Version

// autoSkillRefresh — one-step upgrades: `npm i -g` (or the curl
// installer) is the only step a user needs; the first config-using
// command after a CLI update re-renders any agent skills whose stamp
// predates this build. Silent unless something actually refreshed; dev
// builds and config-less machines do nothing.
func autoSkillRefresh() {
	c, err := config.MustLoad()
	if err != nil {
		return
	}
	names, err := skillinstall.RefreshStale(c)
	if err != nil || len(names) == 0 {
		return
	}
	fmt.Printf("skills refreshed → %s (%s)\n", config.Version, strings.Join(names, ", "))
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	if err := dispatch(args); err != nil {
		fmt.Fprintf(os.Stderr, "log-labor: %v\n", err)
		if ec, ok := err.(exitError); ok {
			os.Exit(ec.code)
		}
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`log-labor ` + version + ` — 报工 into your WeCom 智能表格 labor sheet

  log-labor init                                  configure key + corp id (+ sample row)
  log-labor add -c "内容" -h 2.5 [--date --status --person
                    --proposer --blocker --due --link] [--yes] [--dry-run]
  log-labor update --record-id R [same field flags]
  log-labor doctor [--write-sample]               config + key health, no writes
  log-labor config get|set|list|path [key] [value]   # + default_status|default_proposer|due_mirror
  log-labor profile import [file]                 sheet fields/statuses from 接收外部数据 → 示例数据
  log-labor daily collect                        digest today's agent sessions (all sources)
  log-labor daily sources list|add|remove|test    registry of session stores for collect
  log-labor daily fit "内容"=2.5 …|--total N      fit drafts to sum exactly N (default 8)
  log-labor version

Exit codes: 0 ok · 1 WeCom error · 2 usage/config.
Key + corp id live in ~/.config/log-labor/config.json (0600).
`)
}

func dispatch(args []string) error {
	switch args[0] {
	case "init":
		return cmdInit(args[1:])
	case "add":
		autoSkillRefresh()
		return cmdAdd(args[1:])
	case "update":
		autoSkillRefresh()
		return cmdUpdate(args[1:])
	case "doctor":
		autoSkillRefresh()
		return cmdDoctor(args[1:])
	case "config":
		return cmdConfig(args[1:])
	case "profile":
		return cmdProfile(args[1:])
	case "skill":
		return cmdSkill(args[1:])
	case "daily":
		autoSkillRefresh()
		return cmdDaily(args[1:])
	case "version", "--version", "-v":
		fmt.Println("log-labor " + version)
		return nil
	case "help", "--help", "-h":
		usage()
		return nil
	default:
		usage()
		return exitError{code: 2, msg: fmt.Sprintf("unknown command %q", args[0])}
	}
}
