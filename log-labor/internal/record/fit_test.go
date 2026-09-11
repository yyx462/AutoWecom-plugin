package record

import (
	"math"
	"testing"
)

func eq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestFitToDay(t *testing.T) {
	t.Run("sums to exactly the total", func(t *testing.T) {
		for _, raw := range [][]float64{
			{3, 3, 3}, {1}, {2.5, 2.5, 2.5, 0.5}, {7.3, 0.2, 0.4},
			{0.1, 0.1, 0.1, 6}, {4, 4}, {0.3, 0.3, 0.3, 0.3, 0.3},
		} {
			rows := make([]FitRow, len(raw))
			for i, h := range raw {
				rows[i] = FitRow{Content: "x", Raw: h}
			}
			out, err := FitToDay(rows, 8)
			if err != nil {
				t.Fatalf("%v: %v", raw, err)
			}
			sum := 0.0
			for _, r := range out {
				if math.Mod(r.Fitted, Step) > 1e-9 {
					t.Errorf("%v: fitted %v off-grid", raw, r.Fitted)
				}
				if r.Fitted < Step {
					t.Errorf("%v: fitted %v below floor", raw, r.Fitted)
				}
				sum += r.Fitted
			}
			if !eq(sum, 8) {
				t.Errorf("%v: sum %v != 8", raw, sum)
			}
		}
	})
	t.Run("keeps proportions roughly", func(t *testing.T) {
		out, _ := FitToDay([]FitRow{{Raw: 6}, {Raw: 2}}, 8)
		if !eq(out[0].Fitted, 6) || !eq(out[1].Fitted, 2) {
			t.Errorf("6:2 stayed 6:2, got %v:%v", out[0].Fitted, out[1].Fitted)
		}
		out, _ = FitToDay([]FitRow{{Raw: 6}, {Raw: 1}}, 8)
		if out[0].Fitted <= out[1].Fitted {
			t.Errorf("larger raw should stay larger: %v vs %v", out[0].Fitted, out[1].Fitted)
		}
	})
	t.Run("floor and residual", func(t *testing.T) {
		out, _ := FitToDay([]FitRow{{Raw: 0.1}, {Raw: 7.9}}, 8)
		if out[0].Fitted < Step {
			t.Errorf("tiny work must still get %v", Step)
		}
		out, _ = FitToDay([]FitRow{{Raw: 1}}, 8)
		if !eq(out[0].Fitted, 8) {
			t.Errorf("single record = whole day, got %v", out[0].Fitted)
		}
	})
	t.Run("errors", func(t *testing.T) {
		if _, err := FitToDay(nil, 8); err == nil {
			t.Error("empty rows must error")
		}
		if _, err := FitToDay([]FitRow{{Raw: 0}}, 8); err == nil {
			t.Error("all-zero raw must error")
		}
		if _, err := FitToDay([]FitRow{{Raw: 1}, {Raw: 1}, {Raw: -2}}, 8); err == nil {
			t.Error("negative raw must error")
		}
		many := make([]FitRow, 17)
		for i := range many {
			many[i] = FitRow{Raw: 1}
		}
		if _, err := FitToDay(many, 8); err == nil {
			t.Error("17 records cannot fit 8h at 0.5 granularity")
		}
		if _, err := FitToDay([]FitRow{{Raw: 1}}, 0); err == nil {
			t.Error("total 0 must error")
		}
	})
	t.Run("custom total", func(t *testing.T) {
		out, err := FitToDay([]FitRow{{Raw: 2}, {Raw: 2}}, 4)
		if err != nil || !eq(out[0].Fitted, 2) || !eq(out[1].Fitted, 2) {
			t.Errorf("custom total: %v %v", err, out)
		}
	})
}
