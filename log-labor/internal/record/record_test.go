package record

import (
	"encoding/json"
	"strings"
	"testing"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
)

func testProfile() *config.Profile {
	p := config.DefaultProfile()
	return &p
}

// 1789056000000 = 2026-09-11 00:00 +08 — the value the live webhook
// accepted during the 2026-09-11 probing sessions.
func TestDateToMs(t *testing.T) {
	got, err := DateToMs("2026-09-11")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1789056000000" {
		t.Fatalf("DateToMs(2026-09-11) = %s, want 1789056000000", got)
	}
	if passthrough, _ := DateToMs("1789056000000"); passthrough != "1789056000000" {
		t.Fatalf("bare ms must pass through, got %s", passthrough)
	}
	if _, err := DateToMs("2026/09/11"); err == nil {
		t.Fatal("bad format must error")
	}
}

func TestNormalizeHours(t *testing.T) {
	cases := map[string]string{"3": "3", "2.5": "2.5", "1h": "1", "2小时": "2", " 1.5H ": "1.5"}
	for in, want := range cases {
		got, err := NormalizeHours(in)
		if err != nil || got != want {
			t.Fatalf("NormalizeHours(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := NormalizeHours("abc"); err == nil {
		t.Fatal("non-number hours must error")
	}
}

func TestBuildValues_UserShapes(t *testing.T) {
	p := testProfile()
	// corp id → user field with user_id
	v, err := BuildValues(p, Options{Person: "ying.yuxiang", Date: "2026-09-11",
		Status: "已完成", Content: "x", Hours: "1h", Due: "2026-09-12",
		Proposer: "opencode", Blocker: "无"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if u, ok := v["fQLSz5"].([]map[string]string); !ok || u[0]["user_id"] != "ying.yuxiang" {
		t.Fatalf("person shape wrong: %#v", v["fQLSz5"])
	}
	if v["f3Wcc3"] != "1789056000000" {
		t.Fatalf("date shape wrong: %#v", v["f3Wcc3"])
	}
	if s, ok := v["fFtUk3"].([]map[string]string); !ok || s[0]["text"] != "已完成" {
		t.Fatalf("status shape wrong: %#v", v["fFtUk3"])
	}
	if n, ok := v["fjKRQJ"].(float64); !ok || n != 1 {
		t.Fatalf("hours must be a bare number, got %#v", v["fjKRQJ"])
	}
	if v["fEF4fy"] != "1789142400000" { // 2026-09-12 00:00 +08
		t.Fatalf("due shape wrong: %#v", v["fEF4fy"])
	}
}

func TestBuildValues_RejectsWoaPerson(t *testing.T) {
	// atomic-rejection guard: the CLI must refuse before any HTTP happens
	_, err := BuildValues(testProfile(), Options{Person: "woa-s1CwAAW7x"}, false)
	if err == nil || !strings.Contains(err.Error(), "40031") {
		t.Fatalf("woa- person must be refused client-side, got %v", err)
	}
}

func TestBuildValues_StatusEnum(t *testing.T) {
	if _, err := BuildValues(testProfile(), Options{Status: "废话"}, false); err == nil {
		t.Fatal("off-enum status must error")
	}
}

func TestBuildValues_DefaultsOnlyInAddMode(t *testing.T) {
	p := testProfile()
	add, _ := BuildValues(p, Options{}, true)
	if _, ok := add["f3Wcc3"]; !ok {
		t.Fatal("add mode must default 日期")
	}
	if _, ok := add["fFtUk3"]; !ok {
		t.Fatal("add mode must default 状态")
	}
	upd, _ := BuildValues(p, Options{}, false)
	if len(upd) != 0 {
		t.Fatalf("update mode with no options must stay empty, got %#v", upd)
	}
}

func TestPreviewCJKAlignment(t *testing.T) {
	p := testProfile()
	p.FieldOrder = []string{"person", "date", "status", "content", "hours", "blocker"}
	values := map[string]any{
		"date":    "2026-09-11",
		"content": "[skill验证] 测试（可删除）",
		"hours":   "0.5",
	}
	out := Preview(p, values)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (header/rule/cells), got %d:\n%s", len(lines), out)
	}
	// header and cells must occupy identical display width per column:
	// every line's total display width must match.
	want := dispWidth(lines[0])
	for i, l := range lines {
		if got := dispWidth(l); got != want {
			t.Errorf("line %d width %d != header width %d:\n%s", i, got, want, out)
		}
	}
	if strings.Contains(out, "—") || strings.Contains(out, "|") {
		t.Errorf("old-style placeholders/pipes leaked:\n%s", out)
	}
	if !strings.Contains(lines[2], "·") {
		t.Errorf("unset cells should render as ·:\n%s", out)
	}
}

func TestOptionsFromValuesRoundTrip(t *testing.T) {
	p := testProfile()
	p.FieldOrder = []string{"person", "date", "status", "content", "link", "hours", "proposer", "blocker"}
	in := Options{Person: "zhang.san", Date: "2026-09-11", Status: "已完成",
		Content: "完成登录页联调", Link: "rAbC12,rXyZ98", Hours: "2.5",
		Proposer: "lisi", Blocker: "无"}
	values, err := BuildValues(p, in, false)
	if err != nil {
		t.Fatal(err)
	}
	out := OptionsFromValues(p, values)
	// date legitimately becomes ms-epoch; compare rendered form.
	in.Date, out.Date = CnDate(in.Date), CnDate(out.Date)
	if out != in {
		t.Errorf("round-trip lost data:\n in=%+v\nout=%+v", in, out)
	}
	// JSON-parsed shape must round-trip identically.
	j, _ := json.Marshal(values)
	var parsed map[string]any
	_ = json.Unmarshal(j, &parsed)
	out2 := OptionsFromValues(p, parsed)
	out2.Date = CnDate(out2.Date)
	if out2 != in {
		t.Errorf("json-shape round-trip lost data:\n in=%+v\nout=%+v", in, out2)
	}
}
