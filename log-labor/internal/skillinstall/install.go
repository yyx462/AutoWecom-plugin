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
	if err := writeFile(p, Render(c, "skill")); err != nil {
		return "", err
	}
	return p, nil
}

// InstallProject — one project-relative target (skill folder or stanza).
func InstallProject(c *config.Config, a Agent, kind, rel string) (string, error) {
	p := filepath.Join(rel)
	switch kind {
	case "skill":
		if err := writeFile(filepath.Join(p, "SKILL.md"), Render(c, "skill")); err != nil {
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
