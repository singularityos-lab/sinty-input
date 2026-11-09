package importer

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/singularityos-lab/sinty-input/internal/dict"
)

var compatible = map[string]bool{
	"CC0-1.0": true, "LicenseRef-public-domain": true, "Unlicense": true,
	"MIT": true, "BSD-2-Clause": true, "BSD-3-Clause": true, "ISC": true, "Apache-2.0": true, "Unicode-3.0": true,
	"CC-BY-4.0": true, "CC-BY-SA-4.0": true,
	"GPL-2.0-or-later": true, "GPL-3.0-only": true, "GPL-3.0-or-later": true,
	"LGPL-2.1-or-later": true, "LGPL-3.0-only": true, "LGPL-3.0-or-later": true,
	"LicenseRef-own": true,
}

func Compatible(license string) bool {
	return compatible[license]
}

func Licenses() []string {
	out := make([]string, 0, len(compatible))
	for k := range compatible {
		out = append(out, k)
	}
	return out
}

func Detect(head []byte) string {
	s := string(head)
	switch {
	case strings.Contains(s, "@format "+dict.Format):
		return "sinty"
	case strings.Contains(s, "\n---") || strings.HasPrefix(strings.TrimSpace(s), "---") || strings.Contains(s, "\n...\n"):
		return "rime"
	case strings.Contains(s, "[Data]") || strings.Contains(s, "[数据]") || strings.Contains(s, "KeyCode=") || strings.Contains(s, "键码="):
		return "fcitx"
	}
	return ""
}

func Rime(r io.Reader) (*dict.Dict, error) {
	d := dict.New("", nil)
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1<<16), 1<<22)
	columns := []string{"text", "code", "weight"}
	inHeader, inColumns, inRules := false, false, false
	var rules []string
	var rule string
	body := false
	n := 0
	for s.Scan() {
		n++
		line := strings.TrimRight(s.Text(), "\r")
		if !body {
			t := strings.TrimSpace(line)
			switch {
			case t == "---":
				inHeader = true
				continue
			case t == "...":
				body = true
				if rule != "" {
					rules = append(rules, rule)
				}
				continue
			case !inHeader || t == "" || strings.HasPrefix(t, "#"):
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if indent == 0 {
				inColumns, inRules = false, false
				k, v, _ := strings.Cut(t, ":")
				v = strings.Trim(strings.TrimSpace(v), `"'`)
				switch k {
				case "name":
					d.Meta["rime"] = v
				case "columns":
					columns = nil
					inColumns = true
				}
				continue
			}
			if inColumns && strings.HasPrefix(t, "- ") {
				columns = append(columns, strings.TrimSpace(t[2:]))
				continue
			}
			if strings.HasPrefix(t, "rules:") {
				inRules = true
				continue
			}
			if inRules {
				t = strings.TrimPrefix(t, "- ")
				k, v, _ := strings.Cut(t, ":")
				v = strings.Trim(strings.TrimSpace(v), `"'`)
				switch strings.TrimSpace(k) {
				case "length_equal":
					if rule != "" {
						rules = append(rules, rule)
					}
					rule = v + ":"
				case "length_in_range":
					if rule != "" {
						rules = append(rules, rule)
					}
					lo, _, _ := strings.Cut(strings.Trim(v, "[]"), ",")
					rule = strings.TrimSpace(lo) + "+:"
				case "formula":
					rule += v
				}
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		var text, code string
		weight := 0
		for i, c := range columns {
			if i >= len(fields) {
				break
			}
			switch c {
			case "text":
				text = fields[i]
			case "code":
				code = fields[i]
			case "weight":
				weight = parseWeight(fields[i])
			}
		}
		if text == "" || code == "" {
			continue
		}
		d.Add(code, text, weight)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if !body {
		return nil, fmt.Errorf("no entries after the ... line")
	}
	if len(rules) > 0 {
		d.Meta["phrase"] = strings.Join(rules, " ")
	}
	return d, nil
}

func parseWeight(s string) int {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return int(math.Round(f * 100))
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	if f > math.MaxInt32 {
		return math.MaxInt32
	}
	return int(f)
}

func Fcitx(r io.Reader) (*dict.Dict, error) {
	d := dict.New("", nil)
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	section := ""
	prompt, pinyin, construct := "&", "@", "^"
	var display, rules []string
	for s.Scan() {
		line := strings.TrimRight(s.Text(), "\r")
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, ";") || (section != "data" && strings.HasPrefix(t, "#")) {
			continue
		}
		switch t {
		case "[Data]", "[数据]":
			section = "data"
			continue
		case "[Rule]", "[组词规则]":
			section = "rule"
			continue
		}
		switch section {
		case "":
			k, v, ok := strings.Cut(t, "=")
			if !ok {
				continue
			}
			switch k {
			case "KeyCode", "键码":
				d.Meta["keys"] = v
			case "Length", "码长":
				d.Meta["length"] = v
			case "Prompt", "提示":
				prompt = v
			case "Pinyin", "拼音":
				pinyin = v
			case "ConstructPhrase", "构词":
				construct = v
			}
		case "rule":
			if f, ok := fcitxRule(t); ok {
				rules = append(rules, f)
			}
		case "data":
			i := strings.IndexAny(t, " \t")
			if i < 0 {
				continue
			}
			code, text := t[:i], strings.TrimSpace(t[i+1:])
			switch {
			case prompt != "" && strings.HasPrefix(code, prompt):
				display = append(display, strings.TrimPrefix(code, prompt)+"="+text)
			case pinyin != "" && strings.HasPrefix(code, pinyin), construct != "" && strings.HasPrefix(code, construct):
			case text != "":
				d.Add(code, text, 0)
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if section != "data" {
		return nil, fmt.Errorf("no [Data] section")
	}
	if len(display) > 0 {
		d.Meta["display"] = strings.Join(display, " ")
	}
	if len(rules) > 0 {
		d.Meta["phrase"] = strings.Join(rules, " ")
	}
	return d, nil
}

func fcitxRule(t string) (string, bool) {
	k, v, ok := strings.Cut(t, "=")
	if !ok || len(k) < 2 {
		return "", false
	}
	length := k[1:]
	if k[0] == 'a' {
		length += "+"
	} else if k[0] != 'e' {
		return "", false
	}
	var b strings.Builder
	for _, p := range strings.Split(v, "+") {
		if len(p) != 3 || (p[0] != 'p' && p[0] != 'n') || p[1] < '1' || p[1] > '9' || p[2] < '1' || p[2] > '9' {
			return "", false
		}
		ch := int(p[1] - '1')
		if p[0] == 'p' {
			b.WriteByte(byte('A' + ch))
		} else {
			b.WriteByte(byte('Z' - ch))
		}
		b.WriteByte(byte('a' + p[2] - '1'))
	}
	return length + ":" + b.String(), true
}
