// Package record — build smartsheet values from CLI options + render the
// human preview. All shapes live-proved 2026-09-11:
//
//	text                 plain string                 (readback [{text,type}])
//	number               bare number                  (bare)
//	date_time            ms-epoch STRING              (echoed as sent)
//	single_select        [{"text":opt}] by option text (readback option id)
//	user                 [{"user_id":CORP_ID}]        (woa-/numeric → atomic 40031)
//	two_way_link_records [{"record_id":R}]
package record

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
)

// cst — China Standard Time, fixed +08:00 (no DST → FixedZone is exact and
// needs no tzdata on the host).
var cst = time.FixedZone("CST", 8*3600)

// Options — what one add/update carries (unset ⇒ omitted from the row).
type Options struct {
	Person   string // corp userid; "person" field omitted when empty
	Date     string // YYYY-MM-DD or bare ms
	Status   string
	Content  string
	Link     string // record id(s), comma-separated
	Hours    string // float or trailing-h forms ("3", "2.5", "1h", "2小时")
	Due      string
	Proposer string
	Blocker  string
}

// DateToMs — "2026-09-11" → that day 00:00 +08 as ms STRING; bare ms →
// passthrough (the sheet convention stores the epoch STRING).
func DateToMs(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if isAllDigits(s) {
		return s, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, cst)
	if err != nil {
		return "", fmt.Errorf("bad date %q (want YYYY-MM-DD or epoch-ms)", s)
	}
	return strconv.FormatInt(t.UnixMilli(), 10), nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// NormalizeHours — "3", "2.5", "1h", "2小时" → float string; validates.
func NormalizeHours(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(strings.TrimSuffix(s, "h"), "H")
	s = strings.TrimSuffix(s, "小时")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return "", fmt.Errorf("bad hours %q (want a number, e.g. 2.5)", s)
	}
	return strconv.FormatFloat(f, 'f', -1, 64), nil
}

// CnDate — ms or YYYY-MM-DD → 2026年9月11日 (preview only; best-effort).
func CnDate(s string) string {
	ms, err := DateToMs(s)
	if err != nil || ms == "" {
		return s
	}
	n, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return s
	}
	t := time.UnixMilli(n).In(cst)
	return fmt.Sprintf("%d年%d月%d日", t.Year(), int(t.Month()), t.Day())
}

// BuildValues — typed values map keyed by field id; only the set fields.
// Date defaults to today (+08) when addDefaults is set (add mode).
// A role the profile doesn't map (profile import with an unmatched
// column) disables its flag: unset ⇒ skipped, set ⇒ loud error — never
// a silent write under the empty field id.
func BuildValues(p *config.Profile, o Options, addDefaults bool) (map[string]any, error) {
	f := p.Fields
	v := map[string]any{}
	missing := func(role string) error {
		return fmt.Errorf("profile maps no %s field — cannot set --%s (log-labor profile import)", role, role)
	}

	setUser := func(role, val string) error {
		val = strings.TrimSpace(val)
		if val == "" {
			return nil
		}
		if f[role].ID == "" {
			return missing(role)
		}
		if strings.HasPrefix(val, "woa-") {
			return fmt.Errorf("%s: %q is a bot-namespace id — the webhook rejects it atomically (40031); use the corp userid (e.g. zhang.san)", role, val)
		}
		v[f[role].ID] = []map[string]string{{"user_id": val}}
		return nil
	}
	setDate := func(role, val string) error {
		if strings.TrimSpace(val) == "" {
			return nil
		}
		if f[role].ID == "" {
			return missing(role)
		}
		ms, err := DateToMs(val)
		if err != nil {
			return fmt.Errorf("%s: %w", role, err)
		}
		if ms != "" {
			v[f[role].ID] = ms
		}
		return nil
	}

	if err := setUser("person", o.Person); err != nil {
		return nil, err
	}
	if err := setDate("date", o.Date); err != nil {
		return nil, err
	}
	if o.Status != "" {
		if f["status"].ID == "" {
			return nil, missing("status")
		}
		ok := false
		for _, s := range p.Statuses {
			if s == o.Status {
				ok = true
				break
			}
		}
		if !ok {
			return nil, fmt.Errorf("status must be one of: %s", strings.Join(p.Statuses, " "))
		}
		v[f["status"].ID] = []map[string]string{{"text": o.Status}}
	}
	if o.Content != "" {
		if f["content"].ID == "" {
			return nil, missing("content")
		}
		v[f["content"].ID] = o.Content
	}
	if o.Link != "" {
		if f["link"].ID == "" {
			return nil, missing("link")
		}
		ids := []map[string]string{}
		for _, id := range strings.Split(o.Link, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, map[string]string{"record_id": id})
			}
		}
		if len(ids) > 0 {
			v[f["link"].ID] = ids
		}
	}
	if o.Hours != "" {
		if f["hours"].ID == "" {
			return nil, missing("hours")
		}
		h, err := NormalizeHours(o.Hours)
		if err != nil {
			return nil, fmt.Errorf("hours: %w", err)
		}
		num, _ := strconv.ParseFloat(h, 64)
		v[f["hours"].ID] = num
	}
	if err := setDate("due", o.Due); err != nil {
		return nil, err
	}
	if o.Proposer != "" {
		if f["proposer"].ID == "" {
			return nil, missing("proposer")
		}
		v[f["proposer"].ID] = o.Proposer
	}
	if o.Blocker != "" {
		if f["blocker"].ID == "" {
			return nil, missing("blocker")
		}
		v[f["blocker"].ID] = o.Blocker
	}

	if addDefaults {
		if id := f["date"].ID; id != "" {
			if _, ok := v[id]; !ok {
				today, _ := DateToMs(time.Now().In(cst).Format("2006-01-02"))
				v[id] = today
			}
		}
		if id := f["status"].ID; id != "" {
			if _, ok := v[id]; !ok {
				def := "进行中"
				for _, s := range p.Statuses {
					if s == def {
						goto found
					}
				}
				if len(p.Statuses) > 0 {
					def = p.Statuses[0]
				}
			found:
				v[id] = []map[string]string{{"text": def}}
			}
		}
	}
	return v, nil
}

// firstMap — first element of a webhook array value in either shape:
// []any (as parsed from JSON) or []map[string]string (as built
// in-process by BuildValues). Without both, the preview round-trip
// silently drops person/status/link.
func firstMap(x any) (map[string]string, bool) {
	switch arr := x.(type) {
	case []any:
		if len(arr) > 0 {
			if m, ok := arr[0].(map[string]any); ok {
				out := make(map[string]string, len(m))
				for k, v := range m {
					if s, ok := v.(string); ok {
						out[k] = s
					}
				}
				return out, true
			}
		}
	case []map[string]string:
		if len(arr) > 0 {
			return arr[0], true
		}
	}
	return nil, false
}

// OptionsFromValues — inverse-ish: rebuild Options from a values map
// (profile-driven) so preview + doctor paths share one renderer.
func OptionsFromValues(p *config.Profile, values map[string]any) Options {
	o := Options{}
	get := func(role string) (any, bool) {
		x, ok := values[p.Fields[role].ID]
		return x, ok
	}
	if x, ok := get("person"); ok {
		if m, ok := firstMap(x); ok {
			o.Person = m["user_id"]
		}
	}
	if x, ok := get("date"); ok {
		o.Date = fmt.Sprintf("%v", x)
	}
	if x, ok := get("status"); ok {
		if m, ok := firstMap(x); ok {
			o.Status = m["text"]
		} else if s, ok := x.(string); ok {
			o.Status = s
		}
	}
	if x, ok := get("link"); ok {
		ids := []string{}
		switch arr := x.(type) {
		case []any:
			for _, e := range arr {
				if m, ok := e.(map[string]any); ok {
					if id, ok := m["record_id"].(string); ok {
						ids = append(ids, id)
					}
				}
			}
		case []map[string]string:
			for _, m := range arr {
				if id, ok := m["record_id"]; ok {
					ids = append(ids, id)
				}
			}
		}
		o.Link = strings.Join(ids, ",")
	}
	if x, ok := get("content"); ok {
		if m, ok := firstMap(x); ok {
			o.Content = m["text"]
		} else if s, ok := x.(string); ok {
			o.Content = s
		}
	}
	if x, ok := get("hours"); ok {
		o.Hours = fmt.Sprintf("%v", x)
	}
	if x, ok := get("due"); ok {
		o.Due = fmt.Sprintf("%v", x)
	}
	if x, ok := get("proposer"); ok {
		if m, ok := firstMap(x); ok {
			o.Proposer = m["text"]
		} else if s, ok := x.(string); ok {
			o.Proposer = s
		}
	}
	if x, ok := get("blocker"); ok {
		if m, ok := firstMap(x); ok {
			o.Blocker = m["text"]
		} else if s, ok := x.(string); ok {
			o.Blocker = s
		}
	}
	return o
}

// Preview — the user-facing table (tab-separated, 中文 dates), matching
// the sheet's column order. Only the fields present in values are shown.
// roleNames — profile role → sheet column header (中文).
var roleNames = map[string]string{
	"person": "人员", "date": "日期", "status": "状态",
	"content": "需求内容", "link": "关联", "hours": "预计花费工时",
	"due": "预计完成时间", "proposer": "提出人", "blocker": "卡点",
}

// dispWidth — terminal cells a string occupies: CJK/unambiguous-wide
// runes count 2, everything else 1 (zero-dep East-Asian-width subset,
// enough for this sheet's columns).
func dispWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
			r >= 0x2E80 && r <= 0xA4CF, // CJK radicals … Yi
			r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
			r >= 0xF900 && r <= 0xFAFF, // CJK compat ideographs
			r >= 0xFE30 && r <= 0xFE4F, // CJK compat forms
			r >= 0xFF00 && r <= 0xFF60, // full-width forms
			r >= 0xFFE0 && r <= 0xFFE6, // full-width signs
			r >= 0x20000 && r <= 0x3FFFD: // CJK ext B+
			w += 2
		default:
			w += 1
		}
	}
	return w
}

// padCell — right-pad s with spaces to display width w.
func padCell(s string, w int) string {
	if d := dispWidth(s); d < w {
		return s + strings.Repeat(" ", w-d)
	}
	return s
}

func Preview(p *config.Profile, values map[string]any) string {
	o := OptionsFromValues(p, values)
	headers := make([]string, 0, len(p.FieldOrder))
	cells := make([]string, 0, len(p.FieldOrder))
	for _, role := range p.FieldOrder {
		var s string
		switch role {
		case "person":
			s = o.Person
		case "date":
			s = CnDate(o.Date)
		case "status":
			s = o.Status
		case "content":
			s = o.Content
		case "link":
			s = o.Link
		case "hours":
			s = o.Hours
		case "due":
			s = CnDate(o.Due)
		case "proposer":
			s = o.Proposer
		case "blocker":
			s = o.Blocker
		}
		if s == "" {
			s = "·" // unset — omitted from the write, never written empty
		}
		name := role
		if f, ok := p.Fields[role]; ok && f.Title != "" {
			name = f.Title // mirror the sheet's current column header
		} else if rn := roleNames[role]; rn != "" {
			name = rn
		}
		headers = append(headers, name)
		cells = append(cells, s)
	}
	widths := make([]int, len(headers))
	for i := range headers {
		widths[i] = dispWidth(headers[i])
		if c := dispWidth(cells[i]); c > widths[i] {
			widths[i] = c
		}
	}
	var b strings.Builder
	for i, h := range headers {
		b.WriteString("  " + padCell(h, widths[i]))
	}
	b.WriteString("\n  ")
	for i := range headers {
		b.WriteString(strings.Repeat("─", widths[i]))
		if i < len(headers)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")
	for i, c := range cells {
		b.WriteString("  " + padCell(c, widths[i]))
	}
	b.WriteString("\n")
	return b.String()
}
