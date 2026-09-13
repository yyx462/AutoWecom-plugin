package skillinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
)

func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // windows spelling of UserHomeDir
	return home
}

// opencode is the only agent whose Root (config dir) the test can create
// without faking a whole harness install.
func opencodeAgent(t *testing.T, home string) Agent {
	t.Helper()
	for _, a := range Agents() {
		if a.Name == "opencode" {
			if err := os.MkdirAll(a.Root, 0o755); err != nil {
				t.Fatal(err)
			}
			return a
		}
	}
	t.Fatal("opencode agent not in matrix")
	return Agent{}
}

func writeSkill(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const legacySkill = "---\nname: log-labor\n---\n\n# log-labor — 报工 via the sheet webhook (vv0.1.1)\n\nold body\n"
const stampedSkill = "---\nname: log-labor\n---\n\n# log-labor — 报工 via the sheet webhook (v0.1.1)\n\nold body\n\n<!-- log-labor:v0.1.1 begin (managed by `log-labor skill` — do not edit) -->\n<!-- log-labor end -->\n"

func TestSkillStampLayouts(t *testing.T) {
	home := testHome(t)
	a := opencodeAgent(t, home)

	writeSkill(t, a.Global, stampedSkill)
	if got := SkillStamp(a.Global); got != "v0.1.1" {
		t.Errorf("stamped layout: got %q, want v0.1.1", got)
	}

	writeSkill(t, a.Global, legacySkill)
	if got := SkillStamp(a.Global); got != "v0.1.1" {
		t.Errorf("legacy title layout: got %q, want v0.1.1", got)
	}

	writeSkill(t, a.Global, "---\nname: log-labor\n---\n\n# log-labor — dev build (dev)\n")
	if got := SkillStamp(a.Global); got != "" {
		t.Errorf("dev title must not parse as a version, got %q", got)
	}

	if got := SkillStamp(filepath.Join(home, "nope")); got != "" {
		t.Errorf("missing dir: got %q, want empty", got)
	}
}

func TestSkillStatusAndRefresh(t *testing.T) {
	home := testHome(t)
	a := opencodeAgent(t, home)
	c := &config.Config{Person: "test.user", Profile: config.DefaultProfile()}

	// Mirror production: release builds stamp config.Version via ldflags.
	old := config.Version
	config.Version = "v0.1.2"
	t.Cleanup(func() { config.Version = old })

	// Nothing installed → nothing to do.
	if cur, stale := SkillStatus("v0.1.2"); len(cur)+len(stale) != 0 {
		t.Errorf("empty machine: current=%v stale=%v", cur, stale)
	}
	if names, err := RefreshStale(c); err != nil || len(names) != 0 {
		t.Errorf("empty machine refresh: names=%v err=%v", names, err)
	}

	// Legacy 0.1.1 install + CLI v0.1.2 → stale, refresh re-renders.
	writeSkill(t, a.Global, legacySkill)
	cur, stale := SkillStatus("v0.1.2")
	if len(cur) != 0 || len(stale) != 1 || !strings.HasPrefix(stale[0], "opencode@v0.1.1") {
		t.Fatalf("legacy install: current=%v stale=%v", cur, stale)
	}
	names, err := RefreshStale(c)
	if err != nil || len(names) != 1 || names[0] != "opencode" {
		t.Fatalf("refresh: names=%v err=%v", names, err)
	}
	b, err := os.ReadFile(filepath.Join(a.Global, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "<!-- log-labor:v0.1.2 begin") ||
		!strings.Contains(string(b), "(v0.1.2)") ||
		strings.Contains(string(b), "(vv0.1.1)") {
		t.Errorf("refreshed file not re-stamped:\n%s", b)
	}
	// Frontmatter must still own the first bytes.
	if !strings.HasPrefix(string(b), "---\n") {
		t.Errorf("stamp block must not precede the YAML frontmatter")
	}

	// Now current → SkillStatus clean, refresh is a no-op.
	cur, stale = SkillStatus("v0.1.2")
	if len(cur) != 1 || len(stale) != 0 {
		t.Fatalf("after refresh: current=%v stale=%v", cur, stale)
	}
	if names, err := RefreshStale(c); err != nil || len(names) != 0 {
		t.Errorf("idempotent refresh: names=%v err=%v", names, err)
	}

	// Uninstalled → no stamp → refresh must not resurrect.
	if err := os.RemoveAll(a.Global); err != nil {
		t.Fatal(err)
	}
	if names, err := RefreshStale(c); err != nil || len(names) != 0 {
		t.Errorf("resurrection: names=%v err=%v", names, err)
	}
	if _, err := os.Stat(a.Global); !os.IsNotExist(err) {
		t.Errorf("uninstalled skill came back")
	}
}

func TestRefreshStaleDevBuild(t *testing.T) {
	home := testHome(t)
	a := opencodeAgent(t, home)
	c := &config.Config{Person: "test.user", Profile: config.DefaultProfile()}
	writeSkill(t, a.Global, legacySkill)
	if names, err := RefreshStale(c); err != nil || len(names) != 0 {
		t.Errorf("dev build must never refresh: names=%v err=%v", names, err)
	}
}
