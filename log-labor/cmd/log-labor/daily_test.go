package main

import "testing"

func TestParseTotal(t *testing.T) {
	cases := map[string]float64{
		"":     8,
		"8":    8,
		"10":   10,
		"10h":  10,
		"10H":  10,
		"7.5":  7.5,
		"10小时": 10,
		" 9 ":  9,
	}
	for in, want := range cases {
		got, err := parseTotal(in)
		if err != nil || got != want {
			t.Errorf("parseTotal(%q) = %v, %v; want %v, nil", in, got, err, want)
		}
	}
	for _, in := range []string{"0", "0h", "-4", "abc", "十"} {
		if got, err := parseTotal(in); err == nil {
			t.Errorf("parseTotal(%q) = %v; want error", in, got)
		}
	}
}
