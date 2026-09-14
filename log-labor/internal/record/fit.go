package record

import (
	"fmt"
	"math"
	"sort"
)

// FitRow — one drafted labor record: what was done and the RAW hours
// (the agent's honest estimate). Fitted is filled by FitToDay. Date is
// optional: set on every row to fit several days in one shot (rows
// are grouped per date, each day fitted to the total separately).
type FitRow struct {
	Content string  `json:"content"`
	Raw     float64 `json:"hours"` // stdin field name: hours
	Fitted  float64 `json:"fitted,omitempty"`
	Date    string  `json:"date,omitempty"` // "2006-01-02"; all-or-nothing across rows
}

// Step — the hours granularity: everything snaps to half-hour blocks.
const Step = 0.5

// FitToDay — scale the drafted rows so they sum to EXACTLY total
// (default 8.0 = one working day), keeping proportions and snapping to
// Step granularity. Deterministic: proportional scale → round-half-up
// snap with a 0.5 floor → distribute the residual in Step chunks,
// largest-first (overshoot: smallest-first). The agent drafts honest
// numbers; this makes the day's TOTAL come out right without anyone
// hand-tuning parallel work.
func FitToDay(rows []FitRow, total float64) ([]FitRow, error) {
	if total <= 0 {
		return nil, fmt.Errorf("total must be > 0, got %v", total)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no drafted records")
	}
	if max := int(total/Step + 0.5); len(rows) > max {
		return nil, fmt.Errorf("%d records cannot fit %vh at %vh granularity — merge some first", len(rows), total, Step)
	}
	sum := 0.0
	for _, r := range rows {
		if r.Raw < 0 {
			return nil, fmt.Errorf("negative hours for %q", r.Content)
		}
		sum += r.Raw
	}
	if sum <= 0 {
		return nil, fmt.Errorf("all raw hours are zero — nothing to fit")
	}
	out := make([]FitRow, len(rows))
	copy(out, rows)
	snapped := 0.0
	for i := range out {
		scaled := out[i].Raw * total / sum
		out[i].Fitted = math.Floor(scaled/Step+0.5) * Step
		if out[i].Fitted < Step {
			out[i].Fitted = Step
		}
		snapped += out[i].Fitted
	}
	// Residual — a multiple of Step by construction. Positive:
	// give Step chunks to the largest fitted rows. Negative (rounding
	// overshoot): take them from the smallest (never below the Step
	// floor). len(rows) ≤ total/Step guarantees convergence.
	chunk := Step
	for pass := 0; math.Abs(total-snapped) > 1e-9; pass++ {
		if pass > len(out)+2 {
			return nil, fmt.Errorf("fit failed to converge (residual %v)", total-snapped)
		}
		remaining := total - snapped
		order := make([]int, len(out))
		for i := range order {
			order[i] = i
		}
		if remaining > 0 {
			sort.SliceStable(order, func(a, b int) bool { return out[order[a]].Fitted > out[order[b]].Fitted })
		} else {
			sort.SliceStable(order, func(a, b int) bool { return out[order[a]].Fitted < out[order[b]].Fitted })
		}
		for _, i := range order {
			if math.Abs(remaining) <= 1e-9 {
				break
			}
			if remaining > 0 || out[i].Fitted > Step {
				delta := chunk
				if remaining < 0 {
					delta = -chunk
				}
				out[i].Fitted += delta
				remaining -= delta
				snapped += delta
			}
		}
	}
	for i := range out {
		out[i].Fitted = math.Round(out[i].Fitted*10) / 10 // kill fp dust
	}
	return out, nil
}
