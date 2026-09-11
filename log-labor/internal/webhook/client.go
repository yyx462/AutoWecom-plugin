// Package webhook — the WeDoc smartsheet webhook client.
//
// Live-proved contract (2026-09-11, keep in sync with skill/SKILL.md.tmpl):
//   - POST {endpoint}?key=K  with Content-Type: application/json, UTF-8 body
//   - ONE op per request: a body carrying both add_records and
//     update_records is rejected (40058)
//   - body: {"add_records":[{"values":{FIELD_ID:value}}]}
//     or   {"update_records":[{"record_id":R,"values":{...}}]}
//   - response: {"errcode":0,"errmsg":"ok","add_records|update_records":
//     [{"record_id":"…","values":{…}}]}
//   - error codes seen live: 840001 invalid webhook (bad key) · 2022004
//     field not exists · 2022003 record not exists · 40058 mixed ops ·
//     40031 invalid user_id (user fields accept CORP ids only; rejection
//     is ATOMIC — nothing in the request is written)
//   - rate limits (doc-stated): ≤3000 rows/min per sheet webhook,
//     ≤10000 rows/min per doc across webhooks
package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Record — one add or update op.
type Record struct {
	RecordID string         `json:"record_id,omitempty"`
	Values   map[string]any `json:"values"`
}

// request — exactly one op per body (40058 if mixed).
type request struct {
	AddRecords    []Record `json:"add_records,omitempty"`
	UpdateRecords []Record `json:"update_records,omitempty"`
}

// Response — the envelope; Values kept raw (readback shapes vary by type).
type Response struct {
	Errcode int              `json:"errcode"`
	Errmsg  string           `json:"errmsg"`
	Records []map[string]any `json:"-"`
	raw     json.RawMessage
}

type addResp struct {
	Errcode int            `json:"errcode"`
	Errmsg  string         `json:"errmsg"`
	Add     []map[string]any `json:"add_records"`
	Upd     []map[string]any `json:"update_records"`
}

// RecordID — first record_id in the response ("" when absent).
func (r *Response) RecordID() string {
	for _, m := range r.Records {
		if id, ok := m["record_id"].(string); ok {
			return id
		}
	}
	return ""
}

// errHints — plain-language next steps ("next:") for the errcodes this CLI can
// actually provoke. The key insight for 2022004: the KEY is valid (the
// empty-add probe passed), but the sheet behind it doesn't have the
// profile's field ids — almost always a key pasted from ANOTHER sheet.
var errHints = map[int]string{
	2022004: "this sheet has no such column — if init said \"key accepted\", you likely pasted a key from a DIFFERENT sheet; the key must come from 任务工时详细 → 更多 → 接收外部数据",
	2022003: "record_id not found on this sheet — ids are sheet-scoped; check `log-labor config get sheet`",
	40031:   "bad user value — 人员 needs the corp userid (zhang.san form); woa-… and numeric ids are rejected atomically (nothing written)",
	40058:   "one operation per request — the CLI never mixes add+update; report this if you see it",
	840001:  "invalid webhook key — `log-labor config set key <key>` with the key from 接收外部数据",
}

// Err — non-nil when errcode != 0.
func (r *Response) Err() error {
	if r.Errcode == 0 {
		return nil
	}
	msg := fmt.Sprintf("wecom errcode=%d errmsg=%s", r.Errcode, r.Errmsg)
	if hint, ok := errHints[r.Errcode]; ok {
		msg += " — next: " + hint
	}
	return fmt.Errorf("%s", msg)
}

// Client — endpoint without key; key appended per call (never logged).
type Client struct {
	Endpoint string
	HTTP     *http.Client
}

func New(endpoint string) *Client {
	return &Client{Endpoint: endpoint, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) post(key string, body any) (*Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	u := c.Endpoint + "?key=" + url.QueryEscape(key)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var ar addResp
	if err := json.Unmarshal(raw, &ar); err != nil {
		return nil, fmt.Errorf("unparseable response (http %d): %.200s", resp.StatusCode, raw)
	}
	out := &Response{Errcode: ar.Errcode, Errmsg: ar.Errmsg, raw: raw}
	out.Records = append(out.Records, ar.Add...)
	out.Records = append(out.Records, ar.Upd...)
	return out, nil
}

// AddRecords — ONE add op (values map: field id → typed value).
func (c *Client) AddRecords(key string, values map[string]any) (*Response, error) {
	if len(values) == 0 {
		return nil, errors.New("refusing to add an empty row")
	}
	return c.post(key, request{AddRecords: []Record{{Values: values}}})
}

// UpdateRecords — ONE update op; empty values refused (2022003 otherwise).
func (c *Client) UpdateRecords(key, recordID string, values map[string]any) (*Response, error) {
	if recordID == "" {
		return nil, errors.New("update needs a record id")
	}
	if len(values) == 0 {
		return nil, errors.New("update needs at least one field")
	}
	return c.post(key, request{UpdateRecords: []Record{{RecordID: recordID, Values: values}}})
}

// ProbeKey — key validity WITHOUT writing: an empty add_records array.
// Any errcode other than 840001 (invalid webhook) proves the key was
// accepted; 0 means the no-op succeeded outright.
func (c *Client) ProbeKey(key string) (ok bool, err error) {
	r, err := c.post(key, request{AddRecords: []Record{}})
	if err != nil {
		return false, err
	}
	if r.Errcode == 840001 {
		return false, nil
	}
	return true, nil
}
