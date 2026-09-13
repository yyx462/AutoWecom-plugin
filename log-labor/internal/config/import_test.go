package config

import (
	"reflect"
	"strings"
	"testing"
)

// sampleDoc — the team sheet's 示例数据 exactly as the webhook console
// prints it (2026-09-14). InferProfile must turn this back into
// DefaultProfile — the regression anchor for the heuristics.
const sampleDoc = `{
  "schema": {
    "f3Wcc3": {
      "title": "日期",
      "type": "date_time"
    },
    "fFtUk3": {
      "title": "文本",
      "type": "single_select",
      "enum": [
        "状态",
        "已完成",
        "调休",
        "进行中",
        "待排期",
        "已合并",
        "已关闭"
      ]
    },
    "fQLSz5": {
      "title": "人员",
      "type": "user"
    },
    "fuVenY": {
      "title": "需求内容",
      "type": "text"
    },
    "fzGuQV": {
      "title": "关联",
      "type": "two_way_link_records"
    },
    "fjKRQJ": {
      "title": "预计花费工时",
      "type": "number"
    },
    "fEF4fy": {
      "title": "预计完成时间",
      "type": "date_time"
    },
    "fEyXeR": {
      "title": "提出人",
      "type": "text"
    },
    "f1QYcV": {
      "title": "卡点",
      "type": "text"
    }
  },
  "add_records": [
    {
      "values": {
        "f3Wcc3": "1735660800000",
        "fFtUk3": [{"text": "状态"}],
        "fQLSz5": [{"user_id": ""}],
        "fuVenY": "测试文本",
        "fzGuQV": [{"record_id": ""}],
        "fjKRQJ": 1,
        "fEF4fy": "1735660800000",
        "fEyXeR": "测试文本",
        "f1QYcV": "测试文本"
      }
    }
  ]
}`

func TestParseSampleDoc_FullDoc(t *testing.T) {
	doc, err := ParseSampleDoc([]byte(sampleDoc))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Fields) != 9 {
		t.Fatalf("want 9 fields, got %d", len(doc.Fields))
	}
	if doc.Fields[0].ID != "f3Wcc3" || doc.Fields[0].Title != "日期" {
		t.Fatalf("console field order must be preserved, got %+v first", doc.Fields[0])
	}
	if len(doc.Fields[1].Enum) != 7 || doc.Fields[1].Enum[0] != "状态" {
		t.Fatalf("single_select enum wrong: %#v", doc.Fields[1].Enum)
	}
}

func TestParseSampleDoc_BareSchema(t *testing.T) {
	bare := `{"fX1": {"title": "日期", "type": "date_time"}, "fX2": {"title": "工时", "type": "number"}}`
	doc, err := ParseSampleDoc([]byte(bare))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Fields) != 2 || doc.Fields[0].ID != "fX1" {
		t.Fatalf("bare schema object must parse, got %+v", doc.Fields)
	}
}

func TestParseSampleDoc_BadInput(t *testing.T) {
	if _, err := ParseSampleDoc([]byte("not json")); err == nil {
		t.Fatal("non-JSON must error")
	}
	if _, err := ParseSampleDoc([]byte(`{"a": 1}`)); err == nil {
		t.Fatal("object without schema/title/type must error")
	}
	if _, err := ParseSampleDoc([]byte(`{"schema": []}`)); err == nil {
		t.Fatal("non-object schema must error")
	}
}

func TestInferProfile_TeamSheet(t *testing.T) {
	doc, err := ParseSampleDoc([]byte(sampleDoc))
	if err != nil {
		t.Fatal(err)
	}
	p, warns := InferProfile(doc, Profile{SheetName: "任务工时详细"})
	if len(warns) != 0 {
		t.Fatalf("team sheet should map cleanly, got warnings: %v", warns)
	}
	want := DefaultProfile()
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("team-sheet 示例数据 must reproduce DefaultProfile:\n got  %+v\n want  %+v", p, want)
	}
}

func TestInferProfile_BareSchemaDoc(t *testing.T) {
	// same schema, pasted without the add_records wrapper
	i := strings.Index(sampleDoc, `"schema"`)
	j := strings.Index(sampleDoc, `"add_records"`)
	bare := sampleDoc[i+10 : j]
	bare = strings.TrimSuffix(strings.TrimRight(bare, " \t\r\n"), ",")
	doc, err := ParseSampleDoc([]byte(bare))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := InferProfile(doc, Profile{SheetName: "任务工时详细"})
	if !reflect.DeepEqual(p, DefaultProfile()) {
		t.Fatalf("bare schema must reproduce DefaultProfile too:\n%+v", p)
	}
}

func TestInferProfile_NoSingleSelect(t *testing.T) {
	doc := &SampleDoc{Fields: []FieldSpec{
		{ID: "fA", Title: "人员", Type: "user"},
		{ID: "fB", Title: "工作内容", Type: "text"},
		{ID: "fC", Title: "工时", Type: "number"},
	}}
	p, _ := InferProfile(doc, Profile{SheetName: "x", Statuses: []string{"进行中"}})
	if _, ok := p.Fields["status"]; ok {
		t.Fatal("no single_select → status must stay unmapped")
	}
	if p.Statuses != nil {
		t.Fatalf("statuses must drop to nil without a status field, got %v", p.Statuses)
	}
	for _, role := range []string{"person", "content", "hours"} {
		if p.Fields[role].ID == "" {
			t.Fatalf("%s should have matched", role)
		}
	}
	if len(p.FieldOrder) != 3 {
		t.Fatalf("FieldOrder must list only mapped roles, got %v", p.FieldOrder)
	}
}

func TestInferProfile_EnumCarryover(t *testing.T) {
	doc := &SampleDoc{Fields: []FieldSpec{
		{ID: "fS", Title: "状态", Type: "single_select"}, // no enum
		{ID: "fB", Title: "内容", Type: "text"},
	}}
	prev := Profile{SheetName: "x", Statuses: []string{"进行中", "已完成"}}
	p, warns := InferProfile(doc, prev)
	if !reflect.DeepEqual(p.Statuses, prev.Statuses) {
		t.Fatalf("enum-less single_select must carry previous statuses, got %v", p.Statuses)
	}
	if len(warns) == 0 || !strings.Contains(warns[0], "no enum") {
		t.Fatalf("expected a carryover warning, got %v", warns)
	}
}

func TestApplyMapOverrides(t *testing.T) {
	doc := &SampleDoc{Fields: []FieldSpec{
		{ID: "fA", Title: "工作内容", Type: "text"},
		{ID: "fB", Title: "备注", Type: "text"},
		{ID: "fC", Title: "提示", Type: "text"},
		{ID: "fD", Title: "风险", Type: "text"}, // stays unmapped by inference
		{ID: "fS", Title: "状态", Type: "single_select", Enum: []string{"进行中"}},
	}}
	p, _ := InferProfile(doc, Profile{SheetName: "x"})
	if err := ApplyMapOverrides(&p, doc, "blocker=fD"); err != nil {
		t.Fatal(err)
	}
	if p.Fields["blocker"].ID != "fD" || p.Fields["blocker"].Title != "风险" {
		t.Fatalf("override not applied: %+v", p.Fields["blocker"])
	}
	if err := ApplyMapOverrides(&p, doc, "proposer=fA"); err == nil {
		t.Fatal("double-using a field id must error")
	}
	if err := ApplyMapOverrides(&p, doc, "nope=fB"); err == nil || !strings.Contains(err.Error(), "unknown role") {
		t.Fatalf("unknown role must error, got %v", err)
	}
	if err := ApplyMapOverrides(&p, doc, "blocker=fZZ"); err == nil || !strings.Contains(err.Error(), "no such field id") {
		t.Fatalf("unknown field id must error, got %v", err)
	}
	// re-pointing status must refresh the enum
	doc.Fields = append(doc.Fields, FieldSpec{ID: "fS2", Title: "新状态", Type: "single_select", Enum: []string{"a", "b"}})
	if err := ApplyMapOverrides(&p, doc, "status=fS2"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Statuses, []string{"a", "b"}) {
		t.Fatalf("status override must refresh statuses, got %v", p.Statuses)
	}
}
