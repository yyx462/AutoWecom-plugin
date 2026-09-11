package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/record"
)

// cmdDaily — `daily collect` + `daily fit`: the end-of-day ritual.
// Collect digests today's agent sessions; fit drafts them into records
// summing to exactly one working day (default 8h).
// cnZone — Asia/Shanghai as a fixed +8 zone (sheet's timezone).
func cnZone() *time.Location { return time.FixedZone("CST", 8*3600) }

func cmdDaily(args []string) error {
	if len(args) == 0 {
		return exitError{code: 2, msg: "daily needs a verb: collect|fit"}
	}
	switch args[0] {
	case "collect":
		return dailyCollect(args[1:])
	case "fit":
		return dailyFit(args[1:])
	default:
		return exitError{code: 2, msg: fmt.Sprintf("unknown daily verb %q — want collect|fit", args[0])}
	}
}

// dailyCollect — digest today's opencode sessions from the local
// sqlite store (read-only, via the sqlite3 CLI; zero Go deps).
func dailyCollect(args []string) error {
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	loc := cnZone()
	day := f.val("date")
	if day == "" {
		day = time.Now().In(loc).Format("2006-01-02")
	}
	t0, err := time.ParseInLocation("2006-01-02", day, loc)
	if err != nil {
		return exitError{code: 2, msg: fmt.Sprintf("bad --date %q: %v", day, err)}
	}
	fromMs := t0.UnixMilli()

	db := f.val("db")
	if db == "" {
		home, _ := os.UserHomeDir()
		db = filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	}
	if _, err := os.Stat(db); err != nil {
		return exitError{code: 2, msg: fmt.Sprintf("no opencode store at %s (is opencode installed?)", db)}
	}
	q := fmt.Sprintf(`SELECT m.session_id AS id, MIN(m.time_created) AS t0,
		MAX(m.time_created) AS t1, COUNT(*) AS msgs,
		s.directory AS dir, s.title AS title
		FROM message m JOIN session s ON s.id = m.session_id
		WHERE m.time_created >= %d GROUP BY m.session_id ORDER BY t0;`, fromMs)
	out, err := exec.Command("sqlite3", "-json", "-readonly", db, q).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return exitError{code: 2, msg: fmt.Sprintf("sqlite3: %s", ee.Stderr)}
		}
		return exitError{code: 2, msg: fmt.Sprintf("sqlite3 unavailable or failed: %v", err)}
	}
	var rows []struct {
		ID    string `json:"id"`
		T0    int64  `json:"t0"`
		T1    int64  `json:"t1"`
		Msgs  int    `json:"msgs"`
		Dir   string `json:"dir"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return exitError{code: 2, msg: fmt.Sprintf("parse sqlite output: %v", err)}
	}
	if len(rows) == 0 {
		fmt.Printf("no sessions on %s (+08) — nothing to summarize.\n", day)
		return nil
	}
	fmt.Printf("== sessions on %s (Asia/Shanghai) ==\n", day)
	projects := map[string]bool{}
	for _, r := range rows {
		start := time.UnixMilli(r.T0).In(loc).Format("15:04")
		end := time.UnixMilli(r.T1).In(loc).Format("15:04")
		dur := (r.T1 - r.T0) / 60_000 // ms → minutes
		projects[r.Dir] = true
		fmt.Printf("  %s–%s  %4dm  %3d msgs  %s  %s\n",
			start, end, dur, r.Msgs, r.Dir, r.Title)
	}
	names := make([]string, 0, len(projects))
	for p := range projects {
		names = append(names, filepath.Base(p))
	}
	sort.Strings(names)
	fmt.Printf("%d sessions · projects: %s\n", len(rows), strings.Join(names, ", "))
	fmt.Println("draft records from these (group by task, honest hours), then pipe them to `log-labor daily fit`.")
	return nil
}

// dailyFit — input: drafted records as `daily fit "内容"=2.5 "内容"=1`
// bare args, or a JSON array [{"content":..., "hours":...}] on stdin.
// Output: the fitted table (Σ = --total, default 8) + ready-to-run
// `log-labor add` lines. Never writes to the sheet by itself.
func dailyFit(args []string) error {
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	total := 8.0
	if v := f.val("total"); v != "" {
		if _, err := fmt.Sscanf(v, "%f", &total); err != nil {
			return exitError{code: 2, msg: fmt.Sprintf("bad --total %q", v)}
		}
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
