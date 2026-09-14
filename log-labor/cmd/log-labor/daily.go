package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/record"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/sources"
)

// cmdDaily — `daily collect|sources|fit`: the end-of-day ritual.
// Collect digests the day's agent sessions across every registered or
// built-in store; fit drafts them into records summing to exactly the
// day's total — 8h default, any user-set cap via --total.
func cmdDaily(args []string) error {
	if len(args) == 0 {
		return exitError{code: 2, msg: "daily needs a verb: collect|sources|fit"}
	}
	switch args[0] {
	case "collect":
		return dailyCollect(args[1:])
	case "sources":
		return dailySources(args[1:])
	case "fit":
		return dailyFit(args[1:])
	default:
		return exitError{code: 2, msg: fmt.Sprintf("unknown daily verb %q — want collect|sources|fit", args[0])}
	}
}

// cnZone — Asia/Shanghai as a fixed +8 zone (sheet's timezone).
func cnZone() *time.Location { return time.FixedZone("CST", 8*3600) }

// resolveDay — --date (default today, +08) → start-of-day ms epoch.
func resolveDay(day string) (int64, string, error) {
	if day == "" {
		day = time.Now().In(cnZone()).Format("2006-01-02")
	}
	t0, err := time.ParseInLocation("2006-01-02", day, cnZone())
	if err != nil {
		return 0, "", fmt.Errorf("bad --date %q: %w", day, err)
	}
	return t0.UnixMilli(), day, nil
}

// dailySourcesFor — registered sources + built-ins not shadowed by a
// registration with the same path.
func dailySourcesFor(c *config.Config) []sources.Source {
	out := sources.BuiltIns()
	var keep []sources.Source
	for _, b := range out {
		shadowed := false
		for _, r := range c.Sources {
			if r.Path == b.Path {
				shadowed = true
				break
			}
		}
		if !shadowed {
			keep = append(keep, b)
		}
	}
	return append(keep, c.Sources...)
}

func dailyCollect(args []string) error {
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	fromMs, day, err := resolveDay(f.val("date"))
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	var all []sources.Source
	if db := f.val("db"); db != "" { // legacy override: opencode only
		all = append(all, sources.Source{Name: "opencode", Format: "opencode-sqlite", Path: db})
	} else {
		c, err := config.Load()
		if err == nil {
			all = dailySourcesFor(c)
		} else {
			all = sources.BuiltIns()
		}
	}
	if len(all) == 0 {
		return exitError{code: 2, msg: "no session stores found — register one: `log-labor daily sources add`"}
	}
	ss, warns := sources.CollectAll(all, fromMs)
	for _, w := range warns {
		fmt.Println("  " + w)
	}
	if len(ss) == 0 {
		fmt.Printf("no sessions on %s (+08) — nothing to summarize.\n", day)
		return nil
	}
	fmt.Printf("== sessions on %s (Asia/Shanghai) ==\n", day)
	projects := map[string]bool{}
	for _, r := range ss {
		start := time.UnixMilli(r.Start).In(cnZone()).Format("15:04")
		end := time.UnixMilli(r.End).In(cnZone()).Format("15:04")
		dur := (r.End - r.Start) / 60_000 // ms → minutes
		projects[r.Dir] = true
		title := r.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Printf("  [%s] %s–%s  %4dm  %3d msgs  %s  %s\n",
			r.Source, start, end, dur, r.Msgs, r.Dir, title)
	}
	names := make([]string, 0, len(projects))
	for p := range projects {
		names = append(names, filepath.Base(p))
	}
	sort.Strings(names)
	fmt.Printf("%d sessions · %d sources · projects: %s\n", len(ss), len(all), strings.Join(names, ", "))
	fmt.Println("draft records from these (group by task, honest hours), then pipe them to `log-labor daily fit`.")
	return nil
}

// dailySources — list|add|remove|test: the agent-driven registry that
// teaches log-labor about this machine's session stores.
func dailySources(args []string) error {
	if len(args) == 0 {
		return exitError{code: 2, msg: "sources needs a verb: list|add|remove|test"}
	}
	c, err := config.MustLoad()
	if err != nil {
		return err
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		for _, b := range dailySourcesFor(c) {
			fmt.Printf("  %-12s %-15s %s\n", b.Name, b.Format, b.Path)
		}
		if len(c.Sources) == 0 {
			fmt.Println("(registered: none — built-ins above; add with `daily sources add`)")
		}
		return nil
	case "add":
		f, err := parseFlags(rest)
		if err != nil {
			return err
		}
		s := sources.Source{
			Name:           f.val("name"),
			Format:         f.val("format"),
			Path:           f.val("path"),
			TimestampField: f.val("timestamp-field"),
			CwdField:       f.val("cwd-field"),
			SessionField:   f.val("session-field"),
			TitleField:     f.val("title-field"),
		}
		if s.Name == "" || s.Format == "" || s.Path == "" {
			return exitError{code: 2, msg: "add needs --name, --format, --path"}
		}
		ok := false
		for _, k := range sources.KnownFormats {
			if s.Format == k {
				ok = true
			}
		}
		if !ok {
			return exitError{code: 2, msg: fmt.Sprintf("unknown format %q — want %s", s.Format, strings.Join(sources.KnownFormats, "|"))}
		}
		if p, err := expandHome(s.Path); err == nil {
			s.Path = p
		}
		if _, err := os.Stat(s.Path); err != nil {
			return exitError{code: 2, msg: fmt.Sprintf("no store at %s", s.Path)}
		}
		replaced := false
		for i, old := range c.Sources {
			if old.Name == s.Name {
				c.Sources[i] = s
				replaced = true
			}
		}
		if !replaced {
			c.Sources = append(c.Sources, s)
		}
		if err := c.Save(); err != nil {
			return err
		}
		fmt.Printf("saved (%s: %s @ %s).\n", s.Name, s.Format, s.Path)
		return nil
	case "remove":
		f, err := parseFlags(rest)
		if err != nil {
			return err
		}
		name := f.val("name")
		kept := c.Sources[:0]
		for _, s := range c.Sources {
			if s.Name != name {
				kept = append(kept, s)
			}
		}
		if len(kept) == len(c.Sources) {
			return exitError{code: 2, msg: fmt.Sprintf("no registered source named %q", name)}
		}
		c.Sources = kept
		return c.Save()
	case "test":
		f, err := parseFlags(rest)
		if err != nil {
			return err
		}
		name := f.val("name")
		var target *sources.Source
		for i := range dailySourcesFor(c) {
			if dailySourcesFor(c)[i].Name == name {
				target = &dailySourcesFor(c)[i]
			}
		}
		if target == nil {
			return exitError{code: 2, msg: fmt.Sprintf("no source named %q — see `daily sources list`", name)}
		}
		fromMs, day, err := resolveDay(f.val("date"))
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		ss, warns := sources.CollectAll([]sources.Source{*target}, fromMs)
		for _, w := range warns {
			fmt.Println("  " + w)
		}
		fmt.Printf("[%s] %s: %d sessions on %s\n", target.Name, target.Format, len(ss), day)
		for _, r := range ss {
			fmt.Printf("  %s–%s  %3d msgs  %s  %s\n",
				time.UnixMilli(r.Start).In(cnZone()).Format("15:04"),
				time.UnixMilli(r.End).In(cnZone()).Format("15:04"), r.Msgs, r.Dir, r.Title)
		}
		return nil
	default:
		return exitError{code: 2, msg: fmt.Sprintf("unknown sources verb %q — want list|add|remove|test", verb)}
	}
}

func expandHome(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

// parseTotal — the user-defined day total: "10", "10h", "10小时";
// empty → 8 (one working day). Must be > 0.
func parseTotal(v string) (float64, error) {
	if v == "" {
		return 8, nil
	}
	norm, err := record.NormalizeHours(v)
	if err != nil {
		return 0, fmt.Errorf("bad --total %q (want hours, e.g. 10 / 10h / 10小时)", v)
	}
	f, _ := strconv.ParseFloat(norm, 64)
	if f <= 0 {
		return 0, fmt.Errorf("--total must be > 0, got %q", v)
	}
	return f, nil
}

// dailyFit — input: drafted records as `daily fit "内容"=2.5 "内容"=1`
// bare args, or a JSON array [{"content":..., "hours":...}] on stdin.
// Output: the fitted table (Σ = --total, default 8, user-settable per
// day) + ready-to-run `log-labor add` lines. Never writes to the sheet
// by itself.
func dailyFit(args []string) error {
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	total, err := parseTotal(f.val("total"))
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	var rows []record.FitRow
	if len(f.args) > 0 {
		for _, a := range f.args {
			i := strings.LastIndex(a, "=")
			if i <= 0 {
				return exitError{code: 2, msg: fmt.Sprintf("arg %q is not 内容=hours", a)}
			}
			var h float64
			if _, err := fmt.Sscanf(a[i+1:], "%f", &h); err != nil {
				return exitError{code: 2, msg: fmt.Sprintf("arg %q has bad hours", a)}
			}
			rows = append(rows, record.FitRow{Content: strings.TrimSpace(a[:i]), Raw: h})
		}
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil || len(strings.TrimSpace(string(data))) == 0 {
			return exitError{code: 2, msg: "give records as args (内容=hours) or a JSON array on stdin"}
		}
		if err := json.Unmarshal(data, &rows); err != nil {
			return exitError{code: 2, msg: fmt.Sprintf("stdin is not a records JSON array: %v", err)}
		}
	}
	fitted, err := record.FitToDay(rows, total)
	if err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	fmt.Printf("== 报工草稿 (fit to %.1fh) ==\n", total)
	sum := 0.0
	for i, r := range fitted {
		sum += r.Fitted
		fmt.Printf("  %d. %-4.1fh → %-4.1fh  %s\n", i+1, r.Raw, r.Fitted, r.Content)
	}
	fmt.Printf("Σ = %.1fh\n", sum)
	day := f.val("date")
	dateFlag := ""
	if day != "" {
		dateFlag = " --date " + day
	}
	fmt.Println("-- insert with --yes after the user confirms:")
	for _, r := range fitted {
		fmt.Printf("log-labor add -c %q -h %g%s --yes\n", r.Content, r.Fitted, dateFlag)
	}
	return nil
}
