// Package sources — pluggable session-store collectors for
// `log-labor daily collect`. Built-ins auto-detect when their default
// store exists; anything else is registered by the agent at setup
// (see skill: "Session sources — setup for any agent").
package sources

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Session — one agent session in the digest.
type Session struct {
	Source string `json:"source"` // tag printed in the digest
	Dir    string `json:"dir"`
	Title  string `json:"title"`
	Start  int64  `json:"start"` // ms epoch
	End    int64  `json:"end"`   // ms epoch
	Msgs   int    `json:"msgs"`
	File   string `json:"file,omitempty"` // source jsonl (title fallbacks)
}

// Source — one registered or built-in session store.
type Source struct {
	Name   string `json:"name"`   // digest tag (opencode, claude, …)
	Format string `json:"format"` // opencode-sqlite | jsonl-claude | jsonl-generic
	Path   string `json:"path"`   // db file or directory root

	// jsonl-generic only: which JSON keys hold the signal.
	TimestampField string `json:"timestamp_field,omitempty"`
	CwdField       string `json:"cwd_field,omitempty"`
	SessionField   string `json:"session_field,omitempty"`
	TitleField     string `json:"title_field,omitempty"`
}

// KnownFormats — formats `daily sources add` accepts.
var KnownFormats = []string{"opencode-sqlite", "jsonl-claude", "jsonl-generic"}

// DefaultOpencodeDB — opencode's standard sqlite store.
func DefaultOpencodeDB() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

// DefaultClaudeDir — Claude Code's per-project JSONL store.
func DefaultClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// BuiltIns — the verified defaults, included only when their store
// exists on this machine.
func BuiltIns() []Source {
	var out []Source
	if _, err := os.Stat(DefaultOpencodeDB()); err == nil {
		out = append(out, Source{Name: "opencode", Format: "opencode-sqlite", Path: DefaultOpencodeDB()})
	}
	if st, err := os.Stat(DefaultClaudeDir()); err == nil && st.IsDir() {
		out = append(out, Source{Name: "claude", Format: "jsonl-claude", Path: DefaultClaudeDir()})
	}
	return out
}

// Collect — dispatch by format.
func Collect(s Source, fromMs int64) ([]Session, error) {
	switch s.Format {
	case "opencode-sqlite":
		return CollectOpencode(s.Path, fromMs)
	case "jsonl-claude":
		return CollectClaudeJSONL(s.Path, fromMs)
	case "jsonl-generic":
		return CollectGenericJSONL(s, fromMs)
	default:
		return nil, fmt.Errorf("unknown format %q (want one of %s)", s.Format, strings.Join(KnownFormats, "|"))
	}
}

// CollectAll — run every source, tag results, merge + sort by start.
// Per-source errors degrade to a warning line, never abort the digest.
func CollectAll(sources []Source, fromMs int64) ([]Session, []string) {
	var out []Session
	var warns []string
	for _, s := range sources {
		ss, err := Collect(s, fromMs)
		if err != nil {
			warns = append(warns, fmt.Sprintf("[%s] skipped: %v", s.Name, err))
			continue
		}
		for i := range ss {
			ss[i].Source = s.Name
		}
		out = append(out, ss...)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Start < out[b].Start })
	return out, warns
}

// CollectOpencode — sqlite via the sqlite3 CLI (zero Go deps,
// read-only). One grouped query; ms epoch in/out.
func CollectOpencode(db string, fromMs int64) ([]Session, error) {
	if _, err := os.Stat(db); err != nil {
		return nil, fmt.Errorf("no store at %s", db)
	}
	q := fmt.Sprintf(`SELECT MIN(m.time_created) AS t0, MAX(m.time_created) AS t1,
		COUNT(*) AS msgs, s.directory AS dir, s.title AS title
		FROM message m JOIN session s ON s.id = m.session_id
		WHERE m.time_created >= %d GROUP BY m.session_id ORDER BY t0;`, fromMs)
	out, err := exec.Command("sqlite3", "-json", "-readonly", db, q).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("sqlite3: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("sqlite3 failed: %v", err)
	}
	var rows []struct {
		T0    int64  `json:"t0"`
		T1    int64  `json:"t1"`
		Msgs  int    `json:"msgs"`
		Dir   string `json:"dir"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("parse sqlite output: %v", err)
	}
	ss := make([]Session, 0, len(rows))
	for _, r := range rows {
		ss = append(ss, Session{Dir: r.Dir, Title: r.Title, Start: r.T0, End: r.T1, Msgs: r.Msgs})
	}
	return ss, nil
}

// CollectClaudeJSONL — Claude Code's per-project store: <root>/<munged-
// cwd>/*.jsonl, line-JSON with timestamp (RFC3339), cwd, sessionId.
// Title: first "summary" line, else first user message text.
func CollectClaudeJSONL(root string, fromMs int64) ([]Session, error) {
	summaries := map[string]string{} // file → first summary line
	ss, err := walkJSONL(root, fromMs, func(o map[string]any, acc *acc) {
		if t, _ := o["type"].(string); t == "summary" {
			if _, ok := summaries[acc.File]; !ok {
				summaries[acc.File], _ = o["summary"].(string)
			}
			return
		}
		ms, ok := parseTime(o["timestamp"])
		if !ok {
			return
		}
		acc.see(ms)
		if t2, _ := o["type"].(string); acc.Title == "" && t2 == "user" {
			acc.Title = claudeUserText(o["message"])
		}
	}, func(o map[string]any) string {
		s, _ := o["sessionId"].(string)
		return s
	}, func(o map[string]any) string {
		s, _ := o["cwd"].(string)
		return s
	})
	if err != nil {
		return nil, err
	}
	for i := range ss { // file summary (Claude's own distillation) wins; user text is the fallback
		if summary := summaries[ss[i].File]; summary != "" {
			ss[i].Title = summary
		}
	}
	return ss, nil
}

// CollectGenericJSONL — declared-field line-JSON: the escape hatch for
// "special agents". Timestamp accepts epoch seconds/ms (number or
// numeric string) or RFC3339.
func CollectGenericJSONL(s Source, fromMs int64) ([]Session, error) {
	if s.TimestampField == "" {
		return nil, fmt.Errorf("jsonl-generic needs --timestamp-field")
	}
	return walkJSONL(s.Path, fromMs, func(o map[string]any, acc *acc) {
		ms, ok := parseTime(o[s.TimestampField])
		if !ok {
			return
		}
		acc.see(ms)
		if acc.Title == "" && s.TitleField != "" {
			acc.Title, _ = o[s.TitleField].(string)
		}
	}, func(o map[string]any) string {
		if s.SessionField == "" {
			return "" // one anonymous session per file
		}
		v, _ := o[s.SessionField].(string)
		return v
	}, func(o map[string]any) string {
		if s.CwdField == "" {
			return ""
		}
		v, _ := o[s.CwdField].(string)
		return v
	})
}

// acc — per (file, session) accumulator.
type acc struct {
	Start, End int64
	Msgs       int
	Title      string
	Dir        string
	File       string // source jsonl file (for file-scoped fallbacks)
}

func (a *acc) see(ms int64) {
	if a.Start == 0 || ms < a.Start {
		a.Start = ms
	}
	if ms > a.End {
		a.End = ms
	}
	a.Msgs++
}

// walkJSONL — recurse root for *.jsonl, group lines by sessionKey
// (falling back to the file path when the store has no session ids),
// letting onLine read store-specific fields. Files' ModTime earlier
// than fromMs are skipped unread.
func walkJSONL(root string, fromMs int64,
	onLine func(o map[string]any, a *acc),
	sessionKey func(o map[string]any) string,
	cwd func(o map[string]any) string,
) ([]Session, error) {
	st, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("no store at %s", root)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	type key struct{ file, sess string }
	accs := map[key]*acc{}
	var order []key
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		if fi, err := d.Info(); err != nil || fi.ModTime().UnixMilli() < fromMs {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		perFile := map[string]*acc{}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			var o map[string]any
			if json.Unmarshal([]byte(line), &o) != nil {
				continue
			}
			k := sessionKey(o)
			if _, ok := perFile[k]; !ok {
				perFile[k] = &acc{File: path}
			}
			a := perFile[k]
			onLine(o, a)
			if a.Dir == "" {
				a.Dir = cwd(o)
			}
		}
		for k, a := range perFile {
			if a.Msgs == 0 {
				continue
			}
			id := key{path, k}
			accs[id] = a
			order = append(order, id)
		}
		return nil
	})
	out := make([]Session, 0, len(accs))
	for _, id := range order {
		a := accs[id]
		if a.Start < fromMs {
			continue // session started before the target day
		}
		out = append(out, Session{Dir: a.Dir, Title: a.Title, Start: a.Start, End: a.End, Msgs: a.Msgs, File: a.File})
	}
	return out, nil
}

// parseTime — epoch seconds/ms (number or numeric string) or RFC3339.
func parseTime(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return normEpoch(int64(x)), true
	case string:
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return normEpoch(n), true
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, x); err == nil {
				return t.UnixMilli(), true
			}
		}
	}
	return 0, false
}

func normEpoch(n int64) int64 {
	if n < 1_000_000_000_000 { // seconds
		return n * 1000
	}
	return n
}

// claudeUserText — message.content is a string or [{type:text,text:…}].
func claudeUserText(msg any) string {
	m, ok := msg.(map[string]any)
	if !ok {
		return ""
	}
	switch c := m["content"].(type) {
	case string:
		return truncate(c)
	case []any:
		for _, e := range c {
			if em, ok := e.(map[string]any); ok {
				if t, _ := em["text"].(string); t != "" {
					return truncate(t)
				}
			}
		}
	}
	return ""
}

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > 60 {
		r := []rune(s)
		s = string(r[:60]) + "…"
	}
	return s
}
