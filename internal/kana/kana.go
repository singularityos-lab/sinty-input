package kana

import (
	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/engine"
)

type Engine struct {
	reading string
	pending string
	list    engine.List
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Info() engine.Info {
	return engine.Info{Name: "kana", Label: "Japanese", Symbol: "あ", Language: "ja"}
}

func (e *Engine) text() string {
	return e.reading + e.pending
}

func (e *Engine) show(out engine.Output) {
	t := e.text()
	if !e.list.Empty() {
		t = e.list.Current()
	}
	out.Preedit(t, engine.Runes(t))
	out.Candidates(e.list.Candidates())
}

func candidates(reading string) []string {
	var list []string
	seen := map[string]bool{}
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			list = append(list, s)
		}
	}
	whole := dict.Get("ja-kana").Words(reading)
	var inflected []string
	runes := []rune(reading)
	for n := 1; n <= 4 && n < len(runes); n++ {
		stem, tail := string(runes[:len(runes)-n]), string(runes[len(runes)-n:])
		c := okuri(runes[len(runes)-n])
		if c == "" {
			continue
		}
		for _, w := range dict.Get("ja-kana").Words(stem + c) {
			inflected = append(inflected, w+tail)
		}
	}
	lead, more := min(3, len(whole)), min(6, len(inflected))
	for _, group := range [][]string{whole[:lead], inflected[:more], whole[lead:], inflected[more:]} {
		for _, c := range group {
			add(c)
		}
	}
	if len(list) == 0 {
		add(segment(runes))
	}
	add(reading)
	add(katakana(reading))
	return list
}

func segment(runes []rune) string {
	var out []rune
	for len(runes) > 0 {
		n := min(len(runes), dict.Get("ja-kana").Longest())
		for ; n > 1; n-- {
			if dict.Get("ja-kana").Words(string(runes[:n])) != nil {
				break
			}
		}
		if c := dict.Get("ja-kana").Words(string(runes[:n])); c != nil {
			out = append(out, []rune(c[0])...)
		} else {
			out = append(out, runes[0])
		}
		runes = runes[n:]
	}
	return string(out)
}

func (e *Engine) convert(out engine.Output) {
	e.reading += finish(e.pending)
	e.pending = ""
	e.list.Items = candidates(e.reading)
	e.list.Selected = 0
	e.show(out)
}

func (e *Engine) commit(text string, out engine.Output) {
	if text != "" {
		out.Commit(text)
	}
	e.reading, e.pending = "", ""
	e.list.Clear()
	out.Preedit("", 0)
	out.Candidates(nil)
}

func (e *Engine) converting(k engine.Key, out engine.Output) bool {
	if i, ok := k.Digit(); ok {
		if e.list.Pick(i) {
			e.commit(e.list.Current(), out)
		}
		return true
	}
	switch k.Sym {
	case engine.KeySpace, engine.KeyDown, engine.KeyTab:
		e.list.Move(1)
	case engine.KeyUp:
		e.list.Move(-1)
	case engine.KeyPageDown:
		e.list.Turn(true)
	case engine.KeyPageUp:
		e.list.Turn(false)
	case engine.KeyReturn, engine.KeyKPEnter:
		e.commit(e.list.Current(), out)
		return true
	case engine.KeyEscape, engine.KeyBackSpace:
		e.list.Clear()
	default:
		e.commit(e.list.Current(), out)
		return e.Key(k, out)
	}
	e.show(out)
	return true
}

func (e *Engine) Key(k engine.Key, out engine.Output) bool {
	if k.Release {
		return false
	}
	if k.Shortcut() {
		e.Flush(out)
		return false
	}
	if !e.list.Empty() {
		return e.converting(k, out)
	}
	composing := e.text() != ""
	switch k.Sym {
	case engine.KeySpace:
		if !composing {
			return false
		}
		e.convert(out)
		return true
	case engine.KeyReturn, engine.KeyKPEnter:
		if !composing {
			return false
		}
		e.commit(e.reading+finish(e.pending), out)
		return true
	case engine.KeyEscape:
		if !composing {
			return false
		}
		e.commit("", out)
		return true
	case engine.KeyBackSpace:
		switch {
		case e.pending != "":
			e.pending = e.pending[:len(e.pending)-1]
		case e.reading != "":
			r := []rune(e.reading)
			e.reading = string(r[:len(r)-1])
		default:
			return false
		}
		e.show(out)
		return true
	}
	r, ok := k.Printable()
	if !ok || (r >= '0' && r <= '9' && !composing) {
		e.Flush(out)
		return false
	}
	if r >= 'A' && r <= 'Z' {
		r += 'a' - 'A'
	}
	if _, ok := prefixes[string(r)]; !ok && !(r >= '0' && r <= '9') && e.pending == "" {
		e.Flush(out)
		return false
	}
	kana, rest := convert(e.pending + string(r))
	e.reading += kana
	e.pending = rest
	e.show(out)
	return true
}

func (e *Engine) Select(index int, out engine.Output) {
	if e.list.Pick(index) {
		e.commit(e.list.Current(), out)
	}
}

func (e *Engine) Page(next bool, out engine.Output) {
	e.list.Turn(next)
	e.show(out)
}

func (e *Engine) Reset(out engine.Output) {
	e.commit("", out)
}

func (e *Engine) Flush(out engine.Output) {
	switch {
	case !e.list.Empty():
		e.commit(e.list.Current(), out)
	case e.text() != "":
		e.commit(e.reading+finish(e.pending), out)
	}
}
