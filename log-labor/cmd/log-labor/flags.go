package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

// flags — tiny long/short parser: --name value, --name=value, -c value.
type flags struct {
	vals map[string]string
	set  map[string]bool
	args []string
}

var flagTakesValue = map[string]bool{
	"content": true, "hours": true, "date": true, "status": true,
	"person": true, "proposer": true, "blocker": true, "due": true,
	"link": true, "record-id": true, "key": true, "agent": true, "o": true,
	"sheet": true, "endpoint": true, "db": true, "total": true, "name": true, "format": true,
	"path": true, "timestamp-field": true, "cwd-field": true, "session-field": true,
	"title-field": true,
}
var aliases = map[string]string{
	"c": "content", "h": "hours", "r": "record-id", "y": "yes",
}

func parseFlags(argv []string) (*flags, error) {
	f := &flags{vals: map[string]string{}, set: map[string]bool{}}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			f.args = append(f.args, a)
			continue
		}
		name, val, hasVal := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if long, ok := aliases[name]; ok {
			name = long
		}
		if !flagTakesValue[name] {
			if hasVal {
				return nil, exitError{code: 2, msg: fmt.Sprintf("flag --%s takes no value", name)}
			}
			f.set[name] = true
			f.vals[name] = "true"
			continue
		}
		if !hasVal {
			if i+1 >= len(argv) {
				return nil, exitError{code: 2, msg: fmt.Sprintf("flag --%s needs a value", name)}
			}
			i++
			val = argv[i]
		}
		f.vals[name] = val
		f.set[name] = true
	}
	return f, nil
}

func (f *flags) has(name string) bool   { return f.set[name] }
func (f *flags) val(name string) string { return f.vals[name] }

var stdin = bufio.NewReader(os.Stdin)

// isTTY — best-effort: char device check on stdin.
func isTTY() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// confirm — y/N prompt; non-tty runs must pass --yes.
func confirm(prompt string, forced bool) (bool, error) {
	if forced {
		return true, nil
	}
	if !isTTY() {
		return false, exitError{code: 2, msg: "non-interactive run — re-run with --yes to skip the confirm"}
	}
	fmt.Printf("%s [y/N] ", prompt)
	line, err := stdin.ReadString('\n')
	if err != nil && line == "" { // EOF (closed/null stdin) = unusable confirm
		return false, exitError{code: 2, msg: "no input on confirm (EOF) — re-run with --yes to skip"}
	}
	switch strings.TrimSpace(strings.ToLower(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

// prompt — read a line with a default (EOF/non-tty → default).
func prompt(label, def string) (string, error) {
	if def != "" {
		fmt.Printf("? %s [%s]: ", label, def)
	} else {
		fmt.Printf("? %s: ", label)
	}
	line, err := stdin.ReadString('\n')
	if (err != nil && line == "") || !isTTY() {
		fmt.Println()
		return def, nil
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

// maskKey — first4…last4 (a key is a write credential; never print it whole).
func maskKey(k string) string {
	if len(k) <= 8 {
		return strings.Repeat("*", len(k))
	}
	return k[:4] + "…" + k[len(k)-4:]
}
