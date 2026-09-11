// Package skill — the embedded SKILL.md template (source of truth) and
// its render. skill/SKILL.md.tmpl is the single source; the rendered
// skill/SKILL.md is a checked-in convenience for browsers/registries —
// regenerate with `go run ./cmd/log-labor skill render > skill/SKILL.md`
// (or just `go generate ./...`).
package skill

import (
	_ "embed"
	"strings"
)

//go:generate go run ../../cmd/log-labor skill render -o ../../skill/SKILL.md

//go:embed SKILL.md.tmpl
var raw string

// Vars — injection points; empty values fall back to generic phrasing.
type Vars struct {
	Version       string
	SheetName     string
	Statuses      string // comma-joined
	PersonExample string
}

// Render — fill the template.
func Render(v Vars) string {
	if v.SheetName == "" {
		v.SheetName = "任务工时详细"
	}
	if v.Statuses == "" {
		v.Statuses = "状态, 已完成, 调休, 进行中, 待排期, 已合并, 已关闭"
	}
	if v.PersonExample == "" {
		v.PersonExample = "zhang.san"
	}
	if v.Version == "" {
		v.Version = "dev"
	}
	out := strings.ReplaceAll(raw, "{{VERSION}}", v.Version)
	out = strings.ReplaceAll(out, "{{SHEET}}", v.SheetName)
	out = strings.ReplaceAll(out, "{{STATUSES}}", v.Statuses)
	out = strings.ReplaceAll(out, "{{PERSON_EXAMPLE}}", v.PersonExample)
	return out
}
