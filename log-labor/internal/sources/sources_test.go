package sources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// 2026-09-11 02:00 +08 = 1789092000000 ms; +10min and +30min variants.
const (
	t0ms = 1789092000000
	t1ms = 1789092600000
	t2ms = 1789093800000
)

func TestCollectClaudeJSONL(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "-Users-x-projA", "a.jsonl"),
		`{"type":"summary","summary":"Fixing the webhook","leafUuid":"x"}
{"type":"user","timestamp":"2026-09-11T10:00:00+08:00","cwd":"/Users/x/projA","sessionId":"s1","message":{"content":"hello"}}
{"type":"assistant","timestamp":"2026-09-11T10:10:00+08:00","cwd":"/Users/x/projA","sessionId":"s1","message":{"content":[{"type":"text","text":"hi"}]}}
`)
	ss, err := CollectClaudeJSONL(root, t0ms-3600_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(ss) != 1 {
		t.Fatalf("want 1 session, got %d: %+v", len(ss), ss)
	}
	s := ss[0]
	if s.Msgs != 2 || s.Title != "Fixing the webhook" || s.Dir != "/Users/x/projA" {
		t.Errorf("wrong session: %+v", s)
	}
	if s.Start != t0ms || s.End != t1ms {
		t.Errorf("times: %d..%d", s.Start, s.End)
	}
	// day filter: sessions starting before the boundary are dropped
	ss, _ = CollectClaudeJSONL(root, t0ms+1000)
	if len(ss) != 0 {
		t.Errorf("day filter should drop the session, got %d", len(ss))
	}
}

func TestCollectGenericJSONL(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "s1.jsonl"),
		"{\"ts\": 1789092000, \"project\": \"/p/A\", \"sid\": \"s1\", \"what\": \"epoch seconds\"}\n"+
			"{\"ts\": 1789092600000, \"project\": \"/p/A\", \"sid\": \"s1\", \"what\": \"epoch ms\"}\n")
	mustWrite(t, filepath.Join(root, "s2.jsonl"),
		"{\"ts\": \"2026-09-11T11:50:00+08:00\", \"project\": \"/p/B\", \"sid\": \"s2\", \"what\": \"rfc3339\"}\n")
	s := Source{Name: "g", Format: "jsonl-generic", Path: root,
		TimestampField: "ts", CwdField: "project", SessionField: "sid", TitleField: "what"}
	ss, err := CollectGenericJSONL(s, t0ms-3600_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(ss) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(ss))
	}
	byID := map[string]Session{}
	for _, x := range ss {
		byID[x.Dir] = x
	}
	a, b := byID["/p/A"], byID["/p/B"]
	if a.Msgs != 2 || a.Start != t0ms || a.End != t1ms {
		t.Errorf("epoch session wrong: %+v", a)
	}
	if b.Msgs != 1 || b.Title != "rfc3339" {
		t.Errorf("rfc session wrong: %+v", b)
	}
	if _, err := CollectGenericJSONL(Source{Format: "jsonl-generic", Path: root}, t0ms); err == nil {
		t.Error("missing --timestamp-field must error")
	}
}

func TestCollectAllTagsAndSorts(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "late.jsonl"),
		"{\"ts\": 1789093800000, \"project\": \"/p/B\", \"sid\": \"b\", \"what\": \"late\"}\n")
	mustWrite(t, filepath.Join(root, "early.jsonl"),
		"{\"ts\": 1789092000000, \"project\": \"/p/A\", \"sid\": \"a\", \"what\": \"early\"}\n")
	out, warns := CollectAll([]Source{
		{Name: "g", Format: "jsonl-generic", Path: root,
			TimestampField: "ts", CwdField: "project", SessionField: "sid", TitleField: "what"},
		{Name: "broken", Format: "opencode-sqlite", Path: "/nonexistent/db"},
	}, t0ms-3600_000)
	if len(warns) != 1 || !strings.Contains(warns[0], "[broken]") {
		t.Errorf("broken source must degrade to a warning: %v", warns)
	}
	if len(out) != 2 || out[0].Source != "g" || out[0].Title != "early" || out[1].Title != "late" {
		t.Errorf("merge/sort wrong: %+v", out)
	}
}

func TestKnownFormats(t *testing.T) {
	known := map[string]bool{}
	for _, f := range KnownFormats {
		known[f] = true
	}
	for _, want := range []string{"opencode-sqlite", "jsonl-claude", "jsonl-generic"} {
		if !known[want] {
			t.Errorf("%s missing from KnownFormats", want)
		}
	}
	_, err := Collect(Source{Format: "jsonl-claude", Path: "/nope"}, t0ms)
	if err == nil || strings.Contains(err.Error(), "unknown format") {
		t.Errorf("known format should pass dispatch: %v", err)
	}
	if _, err := Collect(Source{Format: "bogus", Path: "/nope"}, t0ms); err == nil || !strings.Contains(err.Error(), "unknown format") {
		t.Error("bogus format must error with unknown format")
	}
}
