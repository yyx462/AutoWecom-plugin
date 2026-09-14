package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/record"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/skill"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/skillinstall"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/webhook"
)

// sampleContent — the marker convention: webhook cannot delete, so every
// test row announces itself as 可删除 in the row content.
func sampleContent() string {
	return "[skill验证] 测试（可删除）"
}

// resolvePerson — --person > $LOG_LABOR_PERSON > config.
func resolvePerson(c *config.Config, f *flags) string {
	if f.has("person") {
		return strings.TrimSpace(f.val("person"))
	}
	if p := os.Getenv("LOG_LABOR_PERSON"); p != "" {
		return strings.TrimSpace(p)
	}
	return c.Person
}

// applyDefaults — add only: fill unset optionals from config defaults
// (status/proposer) and arm the due mirror. Update stays literal.
func applyDefaults(o record.Options, c *config.Config, add bool) record.Options {
	if !add {
		return o
	}
	o.DueMirror = c.Defaults.MirrorDue()
	if o.Status == "" {
		o.Status = c.Defaults.Status
	}
	if o.Proposer == "" {
		o.Proposer = c.Defaults.Proposer
	}
	return o
}

func optsFromFlags(f *flags, person string) record.Options {
	return record.Options{
		Person:   person,
		Date:     f.val("date"),
		Status:   f.val("status"),
		Content:  f.val("content"),
		Link:     f.val("link"),
		Hours:    f.val("hours"),
		Due:      f.val("due"),
		Proposer: f.val("proposer"),
		Blocker:  f.val("blocker"),
	}
}

func clientFor(c *config.Config) *webhook.Client { return webhook.New(c.Endpoint) }

// writeRow — shared by add/update: build → preview → confirm → POST.
func writeRow(c *config.Config, f *flags, add bool) error {
	var recordID string
	if !add {
		recordID = f.val("record-id")
		if recordID == "" {
			return exitError{code: 2, msg: "update needs --record-id"}
		}
	}
	person := resolvePerson(c, f)
	values, err := record.BuildValues(&c.Profile, applyDefaults(optsFromFlags(f, person), c, add), add)
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	if add && f.val("content") == "" {
		return exitError{code: 2, msg: `add needs --content (and --hours)`}
	}
	if add && f.val("hours") == "" {
		return exitError{code: 2, msg: "add needs --hours"}
	}

	fmt.Print(record.Preview(&c.Profile, values))
	if f.has("dry-run") {
		fmt.Println("dry-run — nothing sent.")
		return nil
	}
	ok, err := confirm(fmt.Sprintf("insert into sheet %q?", c.Profile.SheetName), f.has("yes"))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("aborted — nothing written.")
		return nil
	}

	var resp *webhook.Response
	if add {
		resp, err = clientFor(c).AddRecords(c.Key, values)
	} else {
		resp, err = clientFor(c).UpdateRecords(c.Key, recordID, values)
	}
	if err != nil {
		return exitError{code: 1, msg: err.Error()}
	}
	if err := resp.Err(); err != nil {
		return exitError{code: 1, msg: err.Error()}
	}
	op := "add"
	if !add {
		op = "update"
	}
	fmt.Printf("ok  %s record_id=%s  sheet=%s\n", op, resp.RecordID(), c.Profile.SheetName)
	fmt.Println("(webhook cannot delete — remove rows in the sheet UI if needed)")
	return nil
}

// sampleValues — the marked probe row: today / second status / 0.5h.
func sampleValues(c *config.Config) (map[string]any, error) {
	status := "已完成"
	for _, s := range c.Profile.Statuses {
		if s == "已完成" {
			break
		}
		if len(c.Profile.Statuses) > 1 && s == c.Profile.Statuses[1] {
			status = s
			break
		}
	}
	return record.BuildValues(&c.Profile, record.Options{
		Person:   c.Person,
		Status:   status,
		Content:  sampleContent(),
		Hours:    "0.5",
		Proposer: c.Person,
		Blocker:  "无",
	}, true)
}

func writeSample(c *config.Config, interactive bool) error {
	values, err := sampleValues(c)
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	fmt.Print(record.Preview(&c.Profile, values))
	if interactive {
		ok, err := confirm(fmt.Sprintf("insert this sample row into %q?", c.Profile.SheetName), false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("skipped sample write.")
			return nil
		}
	}
	resp, err := clientFor(c).AddRecords(c.Key, values)
	if err != nil {
		return exitError{code: 1, msg: err.Error()}
	}
	if err := resp.Err(); err != nil {
		return exitError{code: 1, msg: err.Error()}
	}
	fmt.Printf("ok  add record_id=%s — delete this row in the sheet UI when done\n", resp.RecordID())
	return nil
}

// --- commands ---------------------------------------------------------------

func cmdInit(argv []string) error {
	f, err := parseFlags(argv)
	if err != nil {
		return err
	}
	c, err := config.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return exitError{code: 2, msg: err.Error()}
		}
		c = config.Default()
	}
	if !isTTY() && (!f.has("key") || !f.has("person")) {
		return exitError{code: 2, msg: "non-interactive init needs --key and --person"}
	}

	key := f.val("key")
	if key == "" {
		shown := "(none yet)"
		if c.Key != "" {
			shown = maskKey(c.Key)
		}
		key, err = prompt("Webhook key (接收外部数据)", shown)
		if err != nil {
			return err
		}
		if strings.HasPrefix(key, "http") { // pasted the whole URL
			if _, kv, ok := strings.Cut(key, "key="); ok {
				key = strings.TrimSpace(kv)
			}
		}
		if key == shown { // user accepted the masked placeholder = keep old
			key = c.Key
		}
	}
	person := f.val("person")
	if person == "" {
		person, err = prompt("你的企业userid (corp id)", c.Person)
		if err != nil {
			return err
		}
	}
	if strings.HasPrefix(person, "woa-") {
		return exitError{code: 2, msg: fmt.Sprintf("%q is a bot-namespace id — init needs your corp userid (e.g. zhang.san)", person)}
	}
	sheet := f.val("sheet")
	if sheet == "" {
		sheet, err = prompt("Sheet name", c.Profile.SheetName)
		if err != nil {
			return err
		}
	}

	c.Key = strings.TrimSpace(key)
	c.Person = person
	c.Profile.SheetName = sheet
	if err := c.Save(); err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	p, _ := config.Path()
	fmt.Printf("config written to %s (0600)\n", p)

	// prove the pipeline: key probe, then one marked sample row
	ok, err := clientFor(c).ProbeKey(c.Key)
	if err != nil {
		return exitError{code: 1, msg: "key probe failed: " + err.Error()}
	}
	if !ok {
		return exitError{code: 1, msg: "key rejected (840001 invalid webhook) — check `log-labor config set key`"}
	}
	fmt.Printf("key accepted (%s).\n", maskKey(c.Key))
	if f.has("no-sample") {
		return nil
	}
	if !isTTY() && !f.has("yes") {
		fmt.Println("non-interactive: skipping sample write (run `log-labor doctor --write-sample`).")
		return nil
	}
	return writeSample(c, !f.has("yes"))
}

func cmdAdd(argv []string) error {
	f, err := parseFlags(argv)
	if err != nil {
		return err
	}
	c, err := config.MustLoad()
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	return writeRow(c, f, true)
}

func cmdUpdate(argv []string) error {
	f, err := parseFlags(argv)
	if err != nil {
		return err
	}
	c, err := config.MustLoad()
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	return writeRow(c, f, false)
}

func cmdDoctor(argv []string) error {
	f, err := parseFlags(argv)
	if err != nil {
		return err
	}
	c, err := config.MustLoad()
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	fail := false
	p, _ := config.Path()
	fmt.Printf("config   %s\n", p)
	fmt.Printf("endpoint %s\n", c.Endpoint)
	fmt.Printf("sheet    %s\n", c.Profile.SheetName)
	if c.Key == "" {
		fmt.Println("key      MISSING")
		fail = true
	} else {
		fmt.Printf("key      %s\n", maskKey(c.Key))
	}
	switch {
	case c.Person == "":
		fmt.Println("person   MISSING (config set person <corp-id>)")
		fail = true
	case strings.HasPrefix(c.Person, "woa-"):
		fmt.Printf("person   %s  INVALID — bot-namespace id; the webhook rejects it (40031)\n", c.Person)
		fail = true
	default:
		fmt.Printf("person   %s\n", c.Person)
	}
	current, stale := skillinstall.SkillStatus(config.Version)
	switch {
	case len(current)+len(stale) == 0:
		fmt.Println("skills   none installed — `log-labor skill install` gives your agents the skill")
	case len(stale) == 0:
		fmt.Printf("skills   up to date (%s)\n", strings.Join(current, ", "))
	default:
		fmt.Printf("skills   STALE %s (CLI %s) — run: log-labor skill upgrade\n", strings.Join(stale, ", "), config.Version)
	}
	if c.Key != "" {
		ok, err := clientFor(c).ProbeKey(c.Key)
		if err != nil {
			fmt.Printf("key probe  FAILED: %v\n", err)
			fail = true
		} else if !ok {
			fmt.Println("key probe  REJECTED (840001 invalid webhook)")
			fail = true
		} else {
			fmt.Println("key probe  ok")
		}
	}
	if fail {
		return exitError{code: 1, msg: "doctor found problems (see above)"}
	}
	fmt.Println("doctor: all checks passed")
	if f.has("write-sample") {
		return writeSample(c, false)
	}
	return nil
}

func cmdConfig(argv []string) error {
	f, err := parseFlags(argv)
	if err != nil {
		return err
	}
	if f.has("path") || (len(f.args) > 0 && f.args[0] == "path") {
		p, _ := config.Path()
		fmt.Println(p)
		return nil
	}
	if len(f.args) == 0 || f.args[0] == "list" {
		c, err := config.MustLoad()
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		fmt.Printf("endpoint %s\nkey      %s\nperson   %s\nsheet    %s\n", c.Endpoint, maskKey(c.Key), c.Person, c.Profile.SheetName)
		mirror := map[bool]string{true: "on", false: "off"}[c.Defaults.MirrorDue()]
		fmt.Printf("defaults status=%q proposer=%q due_mirror=%s\n", c.Defaults.Status, c.Defaults.Proposer, mirror)
		return nil
	}
	switch f.args[0] {
	case "get":
		if len(f.args) < 2 {
			return exitError{code: 2, msg: "config get <key|person|sheet|endpoint|default_status|default_proposer|due_mirror>"}
		}
		c, err := config.MustLoad()
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		switch f.args[1] {
		case "key":
			fmt.Println(maskKey(c.Key))
		case "person":
			fmt.Println(c.Person)
		case "sheet":
			fmt.Println(c.Profile.SheetName)
		case "endpoint":
			fmt.Println(c.Endpoint)
		case "default_status":
			fmt.Println(c.Defaults.Status)
		case "default_proposer":
			fmt.Println(c.Defaults.Proposer)
		case "due_mirror":
			fmt.Println(map[bool]string{true: "on", false: "off"}[c.Defaults.MirrorDue()])
		default:
			return exitError{code: 2, msg: fmt.Sprintf("unknown config key %q", f.args[1])}
		}
		return nil
	case "set":
		if len(f.args) < 3 {
			return exitError{code: 2, msg: "config set <key|person|sheet|endpoint|default_status|default_proposer|due_mirror> <value>"}
		}
		c, err := config.MustLoad()
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		v := f.args[2]
		switch f.args[1] {
		case "key":
			c.Key = v
		case "person":
			if strings.HasPrefix(v, "woa-") {
				return exitError{code: 2, msg: "person must be a corp userid (e.g. zhang.san), not a woa- id"}
			}
			c.Person = v
		case "sheet":
			c.Profile.SheetName = v
		case "endpoint":
			c.Endpoint = v
		case "default_status":
			if v != "" {
				ok := false
				for _, s := range c.Profile.Statuses {
					if s == v {
						ok = true
						break
					}
				}
				if !ok {
					return exitError{code: 2, msg: fmt.Sprintf("default_status must be one of: %s", strings.Join(c.Profile.Statuses, " "))}
				}
			}
			c.Defaults.Status = v
		case "default_proposer":
			c.Defaults.Proposer = v
		case "due_mirror":
			var b bool
			switch strings.ToLower(v) {
			case "on", "true", "1", "":
				b = true
			case "off", "false", "0":
				b = false
			default:
				return exitError{code: 2, msg: `due_mirror wants on|off`}
			}
			c.Defaults.DueMirror = &b
		default:
			return exitError{code: 2, msg: fmt.Sprintf("unknown config key %q", f.args[1])}
		}
		if err := c.Save(); err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		fmt.Printf("saved (%s).\n", maskKey(c.Key))
		return nil
	default:
		return exitError{code: 2, msg: fmt.Sprintf("unknown config command %q", f.args[0])}
	}
}

// cmdSkill — install|upgrade|uninstall|render.
func cmdSkill(argv []string) error {
	f, err := parseFlags(argv)
	if err != nil {
		return err
	}
	if len(f.args) == 0 {
		return exitError{code: 2, msg: "skill needs a verb: install|upgrade|uninstall|render"}
	}
	verb := f.args[0]
	c, err := config.MustLoad()
	if err != nil && verb != "render" {
		return exitError{code: 2, msg: err.Error() + " — (render works without config)"}
	}

	all := f.has("all")
	want := f.val("agent")
	project := f.has("project")

	selection := func(pred func(skillinstall.Agent) bool) []skillinstall.Agent {
		var out []skillinstall.Agent
		for _, a := range skillinstall.Agents() {
			if pred(a) {
				out = append(out, a)
			}
		}
		return out
	}
	pick := func() ([]skillinstall.Agent, error) {
		switch {
		case all:
			return selection(func(a skillinstall.Agent) bool { return true }), nil
		case want != "":
			return selection(func(a skillinstall.Agent) bool { return a.Name == want }), nil
		default: // detected: harness root exists (project mode also takes rules-only agents)
			return selection(func(a skillinstall.Agent) bool {
				if project && len(a.Project) > 0 {
					return true
				}
				if a.Root == "" {
					return false
				}
				_, err := os.Stat(a.Root)
				return err == nil
			}), nil
		}
	}

	switch verb {
	case "render":
		vars := skill.Vars{Version: config.Version}
		if c != nil {
			vars.SheetName = c.Profile.SheetName
			vars.Statuses = strings.Join(c.Profile.Statuses, ", ")
			vars.PersonExample = c.Person
		}
		out := skill.Render(vars)
		if p := f.val("o"); p != "" { // -o path — used by go:generate for the checked-in copy
			if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
				return exitError{code: 2, msg: err.Error()}
			}
			fmt.Println("rendered →", p)
			return nil
		}
		fmt.Print(out)
		return nil
	case "upgrade", "install":
		targets, err := pick()
		if err != nil {
			return err
		}
		if len(targets) == 0 && !project {
			return exitError{code: 2, msg: "no known agent detected — pass --agent claude|opencode|codex|agents|cursor|trae, --all, or --project"}
		}
		for _, a := range targets {
			if a.Global != "" && !project {
				p, err := skillinstall.InstallGlobal(c, a)
				if err != nil {
					return exitError{code: 2, msg: err.Error()}
				}
				fmt.Printf("installed %-7s → %s\n", a.Name, p)
				continue
			}
			if project {
				for rel, kind := range a.Project {
					p, err := skillinstall.InstallProject(c, a, kind, rel)
					if err != nil {
						return exitError{code: 2, msg: err.Error()}
					}
					fmt.Printf("installed %-7s → %s\n", a.Name, p)
				}
			} else {
				fmt.Printf("skipped  %-7s (project-rules agent — use --project)\n", a.Name)
			}
		}
		if project { // the universal fallbacks
			for _, doc := range []string{"AGENTS.md", "CLAUDE.md"} {
				if err := skillinstall.InstallStanza(c, doc); err != nil {
					return exitError{code: 2, msg: err.Error()}
				}
				fmt.Printf("installed stanza  → %s\n", doc)
			}
		}
		return nil
	case "uninstall":
		targets, err := pick()
		if err != nil {
			return err
		}
		any := false
		for _, a := range targets {
			if a.Global != "" {
				ok, err := skillinstall.UninstallPath(a.Global)
				if err != nil {
					return exitError{code: 2, msg: err.Error()}
				}
				if ok {
					fmt.Printf("removed  %-7s → %s\n", a.Name, a.Global)
					any = true
				}
			}
			if project {
				for rel, kind := range a.Project {
					if kind != "skill" {
						continue
					}
					ok, err := skillinstall.UninstallPath(rel)
					if err != nil {
						return exitError{code: 2, msg: err.Error()}
					}
					if ok {
						fmt.Printf("removed  %-7s → %s\n", a.Name, rel)
						any = true
					}
				}
			}
		}
		if project {
			for _, doc := range []string{"AGENTS.md", "CLAUDE.md"} {
				ok, err := skillinstall.UninstallPath(doc)
				if err != nil {
					return exitError{code: 2, msg: err.Error()}
				}
				if ok {
					fmt.Printf("removed stanza  → %s\n", doc)
					any = true
				}
			}
		}
		if !any {
			fmt.Println("nothing to remove.")
		}
		return nil
	default:
		return exitError{code: 2, msg: fmt.Sprintf("unknown skill verb %q", verb)}
	}
}
