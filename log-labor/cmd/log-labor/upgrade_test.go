package main

import "testing"

func TestNpmManagedPath(t *testing.T) {
	yes := []string{
		"/usr/local/lib/node_modules/@yyx462/log-labor/bin/log-labor",
		"/Users/x/.npm-global/lib/node_modules/@yyx462/darwin-arm64/log-labor",
		"/opt/node/lib/node_modules/log-labor/log-labor",
		"/tmp/npx/@yyx462/log-labor/bin/log-labor",
	}
	no := []string{
		"/usr/local/bin/log-labor",
		"/Users/x/.local/bin/log-labor",
		"C:/tools/log-labor/log-labor.exe",
	}
	for _, p := range yes {
		if !npmManagedPath(p) {
			t.Errorf("want npm-managed: %s", p)
		}
	}
	for _, p := range no {
		if npmManagedPath(p) {
			t.Errorf("want NOT npm-managed: %s", p)
		}
	}
}

func TestNewerVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.1.6", "v0.1.5", true},
		{"0.1.5", "v0.1.6", false},
		{"0.2.0", "0.1.9", true},
		{"1.0.0", "0.9.9", true},
		{"0.1.5", "0.1.5", false},
		{"0.1.5-rc1", "0.1.5", false}, // same core version → not newer
		{"0.2.0-rc1", "0.1.5", true},
		{"garbage", "0.1.5", false},
		{"", "0.1.5", false},
		{"0.1.5", "garbage", false},
	}
	for _, c := range cases {
		if got := newerVersion(c.a, c.b); got != c.want {
			t.Errorf("newerVersion(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
