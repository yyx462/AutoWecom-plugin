// Profile import — derive a sheet Profile from the webhook console's
// 示例数据 document (智能表格 → 右上角文档操作 → 接收外部数据 → 示例数据).
// That console is the only schema source that exists: the webhook is
// write-only, so the pasted sample is ground truth for field ids/types.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// RoleOrder — the canonical roles in display/preview order. Profile.Fields
// is keyed by these; Profile.FieldOrder is this list filtered to the roles
// the sheet actually maps.
var RoleOrder = []string{"date", "status", "person", "content", "link", "hours", "due", "proposer", "blocker"}

// FieldSpec — one schema entry of the 示例数据 document.
type FieldSpec struct {
	ID    string
	Title string
	Type  string // text|number|date_time|single_select|user|two_way_link_records
	Enum  []string
}

// SampleDoc — the parsed 示例数据, field order preserved (decoder-token
// walk; a plain map would scramble it). add_records is deliberately
// dropped: value shapes are role-hardcoded and live-proved in
// internal/record, not derived per sheet.
type SampleDoc struct {
	Fields []FieldSpec
}

// ParseSampleDoc — accept the full {"schema":…,"add_records":…} document
// or a bare schema object ({"fXxx":{"title":…,"type":…}}).
func ParseSampleDoc(b []byte) (*SampleDoc, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, fmt.Errorf("not JSON — want the 接收外部数据 → 示例数据 document: %w", err)
	}
	raw, ok := top["schema"]
	if !ok {
		raw = b // bare schema object
	}
	fields, err := decodeSchema(raw)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, errors.New("no fields found — want 智能表格 → 右上角文档操作 → 接收外部数据 → 示例数据")
	}
	return &SampleDoc{Fields: fields}, nil
}

// decodeSchema — {fieldID:{title,type,enum?}} preserving key order.
func decodeSchema(b []byte) ([]FieldSpec, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	t, err := dec.Token()
	if err != nil || t != json.Delim('{') {
		return nil, errors.New("schema is not a JSON object")
	}
	var out []FieldSpec
	for dec.More() {
		kt, err := dec.Token()
		if err != nil || kt == json.Delim('}') {
			return nil, errors.New("schema is not a JSON object")
		}
		id, ok := kt.(string)
		if !ok {
			return nil, errors.New("schema is not a JSON object")
		}
		var spec struct {
			Title string   `json:"title"`
			Type  string   `json:"type"`
			Enum  []string `json:"enum"`
		}
		if err := dec.Decode(&spec); err != nil {
			return nil, fmt.Errorf("field %s: %v", id, err)
		}
		if spec.Title == "" && spec.Type == "" {
			return nil, fmt.Errorf("field %s has neither title nor type — not a 示例数据 schema", id)
		}
		out = append(out, FieldSpec{ID: id, Title: spec.Title, Type: spec.Type, Enum: spec.Enum})
	}
	return out, nil
}

// InferProfile — map schema fields onto the CLI's roles: title-anchored
// pass first, type fallbacks second, text leftovers fill text roles last.
// Deterministic for a given document. prev supplies SheetName and the
// status fallback when the mapped single_select carries no enum.
func InferProfile(doc *SampleDoc, prev Profile) (Profile, []string) {
	byID := map[string]FieldSpec{}
	for _, fs := range doc.Fields {
		byID[fs.ID] = fs
	}
	taken := map[string]bool{}
	fields := map[string]Field{}
	find := func(pred func(FieldSpec) bool) *FieldSpec {
		for i := range doc.Fields {
			fs := &doc.Fields[i]
			if !taken[fs.ID] && pred(*fs) {
				return fs
			}
		}
		return nil
	}
	hasTitle := func(fs FieldSpec, words ...string) bool {
		for _, w := range words {
			if strings.Contains(fs.Title, w) {
				return true
			}
		}
		return false
	}
	assign := func(role string, fs *FieldSpec) {
		if fs == nil {
			return
		}
		taken[fs.ID] = true
		fields[role] = Field{ID: fs.ID, Title: fs.Title, Type: fs.Type}
	}

	// pass 1 — type + title anchors
	assign("status", find(func(fs FieldSpec) bool { return fs.Type == "single_select" && hasTitle(fs, "状态") }))
	assign("person", find(func(fs FieldSpec) bool { return fs.Type == "user" && hasTitle(fs, "人员", "负责人") }))
	assign("hours", find(func(fs FieldSpec) bool { return fs.Type == "number" && hasTitle(fs, "工时", "小时", "耗时") }))
	assign("link", find(func(fs FieldSpec) bool { return fs.Type == "two_way_link_records" }))
	assign("due", find(func(fs FieldSpec) bool { return fs.Type == "date_time" && hasTitle(fs, "完成", "预计") }))
	assign("date", find(func(fs FieldSpec) bool { return fs.Type == "date_time" && hasTitle(fs, "日期", "时间") }))
	assign("content", find(func(fs FieldSpec) bool { return fs.Type == "text" && hasTitle(fs, "内容", "需求", "事项", "描述") }))
	assign("proposer", find(func(fs FieldSpec) bool { return fs.Type == "text" && hasTitle(fs, "提出人", "申请人") }))
	assign("blocker", find(func(fs FieldSpec) bool { return fs.Type == "text" && hasTitle(fs, "卡点", "阻塞", "风险") }))

	// pass 2 — type fallbacks for the typed roles
	assign("status", find(func(fs FieldSpec) bool { return fs.Type == "single_select" }))
	assign("person", find(func(fs FieldSpec) bool { return fs.Type == "user" }))
	assign("hours", find(func(fs FieldSpec) bool { return fs.Type == "number" }))
	assign("date", find(func(fs FieldSpec) bool { return fs.Type == "date_time" }))
	assign("due", find(func(fs FieldSpec) bool { return fs.Type == "date_time" }))

	// pass 3 — leftover text fields fill unmapped text roles
	for _, role := range []string{"content", "proposer", "blocker"} {
		if fields[role].ID != "" {
			continue // never clobber a pass-1 title match
		}
		assign(role, find(func(fs FieldSpec) bool { return fs.Type == "text" }))
	}

	p := Profile{SheetName: prev.SheetName, Fields: fields}
	var warns []string
	if sf, ok := fields["status"]; ok {
		if enum := byID[sf.ID].Enum; len(enum) > 0 {
			p.Statuses = enum
		} else {
			p.Statuses = prev.Statuses
			warns = append(warns, "status field has no enum in the sample — keeping the previous statuses")
		}
	}
	for _, role := range RoleOrder {
		if f, ok := fields[role]; ok && f.ID != "" {
			p.FieldOrder = append(p.FieldOrder, role)
		}
	}
	var left []string
	for _, fs := range doc.Fields {
		if !taken[fs.ID] {
			left = append(left, fs.Title+" ("+fs.ID+")")
		}
	}
	if len(left) > 0 {
		warns = append(warns, "unmapped schema fields (ignored): "+strings.Join(left, ", "))
	}
	return p, warns
}

// ApplyMapOverrides — "--map role=id[,role=id…]" on top of an inferred
// profile: re-point roles at pasted field ids when the heuristics guess
// wrong. Statuses follow the (possibly new) status field's enum.
func ApplyMapOverrides(p *Profile, doc *SampleDoc, spec string) error {
	byID := map[string]FieldSpec{}
	for _, fs := range doc.Fields {
		byID[fs.ID] = fs
	}
	known := func(role string) bool {
		for _, r := range RoleOrder {
			if r == role {
				return true
			}
		}
		return false
	}
	for _, pair := range strings.Split(spec, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		role, id, ok := strings.Cut(pair, "=")
		role, id = strings.TrimSpace(role), strings.TrimSpace(id)
		if !ok || role == "" || id == "" {
			return fmt.Errorf("--map wants role=id pairs, got %q", pair)
		}
		if !known(role) {
			return fmt.Errorf("--map: unknown role %q (want one of %s)", role, strings.Join(RoleOrder, "|"))
		}
		fs, ok := byID[id]
		if !ok {
			return fmt.Errorf("--map %s=%s: no such field id in the pasted schema", role, id)
		}
		for r, f := range p.Fields {
			if r != role && f.ID == id {
				return fmt.Errorf("--map: field %s is already mapped to %q", id, r)
			}
		}
		if p.Fields == nil {
			p.Fields = map[string]Field{}
		}
		p.Fields[role] = Field{ID: fs.ID, Title: fs.Title, Type: fs.Type}
	}
	if sf, ok := p.Fields["status"]; ok {
		if enum := byID[sf.ID].Enum; len(enum) > 0 {
			p.Statuses = enum
		}
	}
	p.FieldOrder = nil
	for _, role := range RoleOrder {
		if f, ok := p.Fields[role]; ok && f.ID != "" {
			p.FieldOrder = append(p.FieldOrder, role)
		}
	}
	return nil
}
