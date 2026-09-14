package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"git.sh.nint.com/ying.yuxiang/AutoWecom-plugin/log-labor/internal/config"
)

// cmdProfile — manage the sheet profile. The webhook console's 示例数据
// is the only schema source (the webhook is write-only), so import is
// the one verb that matters.
func cmdProfile(argv []string) error {
	if len(argv) == 0 {
		return exitError{code: 2, msg: "profile needs a verb: import"}
	}
	switch argv[0] {
	case "import":
		return cmdProfileImport(argv[1:])
	default:
		return exitError{code: 2, msg: fmt.Sprintf("unknown profile verb %q (want import)", argv[0])}
	}
}

// cmdProfileImport — log-labor profile import [file] [--map role=id,…]
// [--sheet NAME] [--yes]. Reads the 示例数据 from a file, stdin pipe, or
// an interactive paste; infers the role→field map + statuses; shows the
// mapping and asks before writing.
func cmdProfileImport(argv []string) error {
	f, err := parseFlags(argv)
	if err != nil {
		return err
	}
	if len(f.args) > 1 {
		return exitError{code: 2, msg: "usage: profile import [file] [--map role=id,…] [--sheet NAME] [--yes]"}
	}

	var doc *config.SampleDoc
	if len(f.args) == 1 {
		b, err := os.ReadFile(f.args[0])
		if err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
		if doc, err = config.ParseSampleDoc(b); err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
	} else if !isTTY() {
		b, err := io.ReadAll(os.Stdin)
		if err != nil || len(strings.TrimSpace(string(b))) == 0 {
			return exitError{code: 2, msg: "paste the 示例数据 JSON on stdin, or pass a file: log-labor profile import <sample.json>"}
		}
		if doc, err = config.ParseSampleDoc(b); err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
	} else {
		for attempt := 1; ; attempt++ {
			txt, err := promptMultiline("粘贴 示例数据 JSON — 智能表格 → 右上角文档操作 → 接收外部数据 → 示例数据")
			if err != nil {
				return err
			}
			if doc, err = config.ParseSampleDoc([]byte(txt)); err == nil {
				break
			}
			fmt.Println("parse failed: " + err.Error())
			if attempt >= 3 {
				return exitError{code: 2, msg: "giving up — save the JSON to a file and run: log-labor profile import <file>"}
			}
		}
	}

	c, err := config.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return exitError{code: 2, msg: err.Error()}
		}
		c = config.Default() // fresh machine: profile before init
	}
	prof, warns := config.InferProfile(doc, c.Profile)
	if m := f.val("map"); m != "" {
		if err := config.ApplyMapOverrides(&prof, doc, m); err != nil {
			return exitError{code: 2, msg: err.Error()}
		}
	}
	if len(prof.FieldOrder) == 0 {
		return exitError{code: 2, msg: "no schema field matched any role — check the document or use --map role=id"}
	}
	sheet := f.val("sheet")
	if sheet == "" && isTTY() {
		if sheet, err = prompt("Sheet name", c.Profile.SheetName); err != nil {
			return err
		}
	}
	if sheet != "" {
		prof.SheetName = sheet
	}

	fmt.Printf("sheet    %s\n", prof.SheetName)
	for _, role := range config.RoleOrder {
		if fld, ok := prof.Fields[role]; ok && fld.ID != "" {
			fmt.Printf("  %-9s → %s (%s, %s)\n", role, fld.Title, fld.ID, fld.Type)
		} else {
			fmt.Printf("  %-9s → (unmapped — --%s disabled)\n", role, role)
		}
	}
	if len(prof.Statuses) > 0 {
		fmt.Printf("statuses %s\n", strings.Join(prof.Statuses, ", "))
	} else {
		fmt.Println("statuses (none — --status disabled)")
	}
	for _, w := range warns {
		fmt.Println("note     " + w)
	}

	ok, err := confirm("write this profile?", f.has("yes"))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("aborted — profile unchanged.")
		return nil
	}
	c.Profile = prof
	if err := c.Save(); err != nil {
		return exitError{code: 2, msg: err.Error()}
	}
	p, _ := config.Path()
	fmt.Printf("profile written to %s\n", p)
	fmt.Println("prove it against the sheet: log-labor doctor --write-sample")
	return nil
}
