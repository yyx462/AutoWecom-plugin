// Package skillinstall — install the rendered skill for every agent
// harness we know. Two mechanisms:
//
//  1. SKILL.md folders (Agent Skills format) — Claude Code, opencode,
//     Codex, and the generic ~/.agents/skills convention.
//  2. Rules/stanza fallback for harnesses without skill folders — Cursor
//     (.cursor/rules/*.mdc), Trae (.trae/rules/*.md), and a marked
//     AGENTS.md / CLAUDE.md block that nearly every agent reads.
//
// Installs are idempotent and stamped with the CLI version; upgrade =
// install; uninstall only touches files we own.
package skillinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/skill"
)

// Agent — one harness target.
type Agent struct {
	Name    string            // claude|opencode|codex|agents|cursor|trae
	Root    string            // harness root dir — existence = "installed here" ("" = none)
	Global  string            // absolute dir for SKILL.md installs ("" = stanza/project only)
	Desc    string            // human label
	Stanza  bool              // rules-stanza style instead of SKILL.md folder
	Project map[string]string // project-relative path → renderer kind
}

const begin = "<!-- log-labor:%s begin (managed by `log-labor skill` — do not edit) -->"
const end = "<!-- log-labor end -->"

// stampBlock — trailing ownership+version marker for SKILL.md folder
// installs. It must sit AFTER the YAML frontmatter (which has to own the
// first bytes of the file for harnesses to parse it), so trailing HTML
// comments carry the stamp; the AGENTS.md stanza keeps its leading wrap.
// SkillStamp reads both layouts.
func stampBlock(version string) string {
	return fmt.Sprintf("\n%s\n%s\n", fmt.Sprintf(begin, version), end)
}

// Agents — the full matrix. Global paths resolve lazily via home.
func Agents() []Agent {
	home, _ := os.UserHomeDir()
	j := filepath.Join
	// opencode's global config lives under XDG (~/.config/opencode) on unix
	// but %AppData%\opencode on windows; prefer whichever exists.
	ocRoot := j(home, ".config", "opencode")
	if runtime.GOOS == "windows" {
		if d, err := os.UserConfigDir(); err == nil {
			if _, err := os.Stat(j(d, "opencode")); err == nil {
				ocRoot = j(d, "opencode")
			}
		}
	}
	return []Agent{
		{Name: "claude", Desc: "Claude Code", Root: j(home, ".claude"), Global: j(home, ".claude", "skills", "log-labor"),
			Project: map[string]string{".claude/skills/log-labor": "skill"}},
		{Name: "opencode", Desc: "opencode", Root: ocRoot, Global: j(ocRoot, "skills", "log-labor"),
			Project: map[string]string{".opencode/skills/log-labor": "skill"}},
		{Name: "codex", Desc: "Codex CLI", Root: j(home, ".codex"), Global: j(home, ".codex", "skills", "log-labor"),
			Project: map[string]string{".codex/skills/log-labor": "skill"}},
		{Name: "agents", Desc: "generic ~/.agents/skills", Root: j(home, ".agents"), Global: j(home, ".agents", "skills", "log-labor"),
			Project: map[string]string{".agents/skills/log-labor": "skill"}},
		{Name: "cursor", Desc: "Cursor (project rules)", Stanza: true,
			Project: map[string]string{".cursor/rules/log-labor.mdc": "mdc"}},
		{Name: "trae", Desc: "Trae (project rules)", Stanza: true,
			Project: map[string]string{".trae/rules/log-labor.md": "rules"}},
	}
}

// Render — the skill body for one target kind.
func Render(c *config.Config, kind string) string {
	body := skill.Render(skill.Vars{
		Version:       config.Version,
		SheetName:     c.Profile.SheetName,
		Statuses:      strings.Join(c.Profile.Statuses, ", "),
		PersonExample: c.Person,
	})
	switch kind {
	case "mdc":
		return fmt.Sprintf("---\ndescription: 报工/labor logging via the log-labor CLI (%s smartsheet webhook). Use for 记工时/报工 asks.\nalwaysApply: false\n---\n\n%s", c.Profile.SheetName, body)
	case "rules":
		return body
	default: // "skill"
		return body
	}
}

// StanzaBody — condensed rules block for AGENTS.md / CLAUDE.md.
func StanzaBody(c *config.Config) string {
	return fmt.Sprintf(`## log-labor (报工 — %s smartsheet)

- log labor on demand: log-labor add -c "ONE-sentence summary of the work" -h HOURS [--status --date --proposer --blocker --due --link]
- closure/rework = log-labor update --record-id R …, NEVER a second row
- confirm ambiguous hours with the user; report the record_id after writing
- 人员 needs the corp userid (zhang.san form) — woa- ids are atomically rejected
- doctor --write-sample = one marked 可删除 test row; the webhook cannot delete
`, c.Profile.SheetName)
}

func wrap(version string, body string) string {
	return fmt.Sprintf(begin+"\n\n%s\n"+end+"\n", version, body)
}

// findStanza — start/end offsets of our block in a doc ("" if absent).
func findStanza(doc, version string) (int, int) {
	b := strings.Index(doc, fmt.Sprintf(begin, version))
	if b < 0 { // older version stamps still match on the end marker pair
		b = strings.Index(doc, "<!-- log-labor:")
		if b < 0 {
			return -1, -1
		}
	}
	e := strings.Index(doc[b:], end)
	if e < 0 {
		return -1, -1
	}
	return b, b + e + len(end)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// InstallGlobal — SKILL.md into one agent's global dir.
func InstallGlobal(c *config.Config, a Agent) (string, error) {
	if a.Global == "" {
		return "", fmt.Errorf("%s has no global skill dir (use --project)", a.Name)
	}
	p := filepath.Join(a.Global, "SKILL.md")
	if err := writeFile(p, Render(c, "skill")+stampBlock(config.Version)); err != nil {
		return "", err
	}
	return p, nil
}

// InstallProject — one project-relative target (skill folder or stanza).
func InstallProject(c *config.Config, a Agent, kind, rel string) (string, error) {
	p := filepath.Join(rel)
	switch kind {
	case "skill":
		if err := writeFile(filepath.Join(p, "SKILL.md"), Render(c, "skill")+stampBlock(config.Version)); err != nil {
			return "", err
		}
		return filepath.Join(p, "SKILL.md"), nil
	case "mdc", "rules":
		return p, writeFile(p, Render(c, kind))
	default: // stanza
		old, _ := os.ReadFile(p)
		doc := string(old)
		if s, e := findStanza(doc, ".*"); s >= 0 {
			doc = doc[:s] + doc[e:]
		}
		doc = strings.TrimRight(doc, "\n")
		if doc != "" {
			doc += "\n\n"
		}
		return p, writeFile(p, doc+wrap(config.Version, StanzaBody(c)))
	}
}

// InstallStanza — AGENTS.md/CLAUDE.md marked block (project or explicit path).
func InstallStanza(c *config.Config, path string) error {
	_, err := InstallProject(c, Agent{Stanza: true}, "stanza", path)
	return err
}

// legacyTitleRe — pre-stamp installs carried the version only in the
// title line: `# log-labor — … (vv0.1.1)` (the template had a literal
// `v` next to the version, hence the v-run).
var legacyTitleRe = regexp.MustCompile(`\((v+[0-9][^)]*)\)`)

// SkillStamp — the CLI version an installed skill folder was rendered by
// ("" = not ours / unreadable). Two layouts: the trailing stamp block
// written since auto-refresh landed, and the legacy title line.
func SkillStamp(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return ""
	}
	s := string(b)
	const mark = "<!-- log-labor:"
	if i := strings.Index(s, mark); i >= 0 {
		rest := s[i+len(mark):]
		if j := strings.Index(rest, " begin"); j >= 0 {
			return rest[:j]
		}
	}
	for _, line := range strings.Split(s, "\n") {
		if !strings.HasPrefix(line, "# log-labor") {
			continue
		}
		if m := legacyTitleRe.FindStringSubmatch(line); m != nil {
			return "v" + strings.TrimLeft(m[1], "v")
		}
	}
	return ""
}

// SkillStatus — names of the global skill folders installed on this
// machine, split into current (stamp matches version) and stale. A dev
// build never reports stale (its stamps are release versions).
func SkillStatus(version string) (current, stale []string) {
	for _, a := range Agents() {
		if a.Global == "" {
			continue
		}
		st := SkillStamp(a.Global)
		if st == "" {
			continue
		}
		if version != "dev" && st != version {
			stale = append(stale, fmt.Sprintf("%s@%s", a.Name, st))
			continue
		}
		current = append(current, a.Name)
	}
	return current, stale
}

// RefreshStale — re-render the global skills whose version stamp
// predates the running build (config.Version), for harnesses present on
// this machine. Returns the agent names refreshed. Skills never
// installed — or uninstalled, which leaves no stamp — are not
// resurrected; dev builds never refresh.
func RefreshStale(c *config.Config) ([]string, error) {
	if config.Version == "dev" {
		return nil, nil
	}
	var out []string
	for _, a := range Agents() {
		if a.Global == "" {
			continue
		}
		if _, err := os.Stat(a.Root); err != nil {
			continue // harness not installed here
		}
		st := SkillStamp(a.Global)
		if st == "" || st == config.Version {
			continue
		}
		if _, err := InstallGlobal(c, a); err != nil {
			return out, err
		}
		out = append(out, a.Name)
	}
	return out, nil
}

// UninstallPath — remove our block from a stanza doc, or delete a skill
// folder/file we own (only if it carries our first line).
func UninstallPath(path string) (bool, error) {
	st, err := os.Stat(path)
	if err != nil {
		return false, nil
	}
	if st.IsDir() {
		sk := filepath.Join(path, "SKILL.md")
		b, err := os.ReadFile(sk)
		if err != nil || !strings.Contains(string(b), "log-labor") {
			return false, nil
		}
		return true, os.RemoveAll(path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	doc := string(b)
	if !strings.Contains(doc, "<!-- log-labor:") {
		return false, nil
	}
	s, e := findStanza(doc, ".*")
	if s < 0 {
		return false, nil
	}
	doc = strings.TrimRight(doc[:s]+doc[e:], "\n") + "\n"
	return true, os.WriteFile(path, []byte(doc), 0o644)
}
