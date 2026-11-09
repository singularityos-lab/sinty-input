package table

import (
	"strings"
	"unicode/utf8"

	"github.com/singularityos-lab/sinty-input/internal/convert"
	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/punct"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

const (
	completions = 40
	maxItems    = 300
)

type Scheme struct {
	Name   string
	Label  string
	Symbol string
	Dict   string
}

type Engine struct {
	scheme   Scheme
	opts     engine.Options
	d        *dict.Dict
	user     *userdict.Store
	keys     string
	length   int
	wildcard rune
	auto     bool
	display  map[rune]string
	code     string
	list     engine.List
}

func New(s Scheme) *Engine {
	e := &Engine{scheme: s, opts: engine.DefaultOptions()}
	e.user = userdict.Open(s.Name)
	return e
}

func (e *Engine) dict() *dict.Dict {
	name := e.scheme.Dict
	if e.opts.Dictionary != "" {
		name = e.opts.Dictionary
	}
	d := dict.Get(name)
	if d != e.d {
		e.d = d
		e.keys = d.Get("keys")
		e.length = d.Int("length")
		if e.length <= 0 {
			e.length = d.Longest()
		}
		e.wildcard, _ = utf8.DecodeRuneInString(d.Get("wildcard"))
		e.auto = d.Get("autocommit") == "yes"
		e.display = map[rune]string{}
		for _, pair := range strings.Fields(d.Get("display")) {
			k, v, ok := strings.Cut(pair, "=")
			if r, size := utf8.DecodeRuneInString(k); ok && size == len(k) && v != "" {
				e.display[r] = v
			}
		}
	}
	return d
}

func (e *Engine) Language() string {
	if l := e.dict().Get("language"); l != "" {
		return l
	}
	return "zh"
}

func (e *Engine) Info() engine.Info {
	label := e.scheme.Label
	if label == "" {
		label = e.dict().Get("name")
	}
	return engine.Info{Name: e.scheme.Name, Label: label, Symbol: e.scheme.Symbol, Language: e.Language()}
}

func (e *Engine) Available() bool {
	return e.dict() != nil
}

func (e *Engine) Configure(o engine.Options) {
	e.opts = o
}

func (e *Engine) Kind() string {
	return "table"
}

func (e *Engine) Dictionary() string {
	if e.opts.Dictionary != "" {
		return e.opts.Dictionary
	}
	return e.scheme.Dict
}

func (e *Engine) Options() engine.Options {
	return e.opts
}

func (e *Engine) key(r rune) bool {
	return r < 0x80 && strings.ContainsRune(e.keys, r) || (e.wildcard != 0 && r == e.wildcard && !strings.ContainsRune(e.keys, r) && e.code != "")
}

func (e *Engine) shown() string {
	var b strings.Builder
	for _, r := range e.code {
		if v, ok := e.display[r]; ok {
			b.WriteString(v)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func match(pattern, code string, wildcard rune) bool {
	p, c := []rune(pattern), []rune(code)
	if len(p) != len(c) {
		return false
	}
	for i := range p {
		if p[i] != wildcard && p[i] != c[i] {
			return false
		}
	}
	return true
}

func (e *Engine) candidates(code string) []string {
	d := e.dict()
	if e.wildcard != 0 && strings.ContainsRune(code, e.wildcard) {
		prefix := code[:strings.IndexRune(code, e.wildcard)]
		var out []string
		seen := map[string]bool{}
		for _, c := range d.Prefixed(prefix, 0) {
			if !match(code, c, e.wildcard) {
				continue
			}
			for _, t := range userdict.Rank(c, d.Lookup(c), e.user) {
				if !seen[t] {
					seen[t] = true
					out = append(out, t)
				}
			}
			if len(out) >= maxItems {
				break
			}
		}
		return out
	}
	out := userdict.Rank(code, d.Lookup(code), e.user)
	seen := map[string]bool{}
	for _, t := range out {
		seen[t] = true
	}
	if len([]rune(code)) < e.length {
		for _, c := range d.Prefixed(code, completions+1) {
			if c == code {
				continue
			}
			for _, x := range d.Lookup(c)[:min(2, len(d.Lookup(c)))] {
				if !seen[x.Text] {
					seen[x.Text] = true
					out = append(out, x.Text)
				}
			}
			if len(out) >= maxItems {
				break
			}
		}
	}
	return out
}

func (e *Engine) refresh() {
	e.list.Clear()
	e.list.Items = e.candidates(e.code)
	e.list.Aux = e.shown()
}

func (e *Engine) show(out engine.Output) {
	if e.code == "" {
		out.Preedit("", 0)
		out.Candidates(nil)
		return
	}
	t := e.shown()
	out.Preedit(t, engine.Runes(t))
	out.Candidates(e.list.Candidates())
}

func (e *Engine) emit(text string, out engine.Output) {
	if text == "" {
		return
	}
	out.Commit(convert.Apply(text, convert.Chain(e.Language(), e.opts.Conversion)))
}

func (e *Engine) clear(out engine.Output) {
	e.code = ""
	e.list.Clear()
	out.Preedit("", 0)
	out.Candidates(nil)
}

func (e *Engine) choose(out engine.Output) {
	if e.list.Empty() {
		e.clear(out)
		return
	}
	text, code := e.list.Current(), e.code
	if e.opts.Learn && !strings.ContainsRune(code, e.wildcard) {
		if _, exact := e.dict().Weight(code, text); exact {
			e.user.Learn(code, text)
		}
	}
	e.clear(out)
	e.emit(text, out)
}

func (e *Engine) navigate(k engine.Key, out engine.Output) bool {
	switch k.Sym {
	case engine.KeyDown:
		e.list.Move(1)
	case engine.KeyUp:
		e.list.Move(-1)
	case engine.KeyPageDown, engine.KeyRight:
		e.list.Turn(true)
	case engine.KeyPageUp, engine.KeyLeft:
		e.list.Turn(false)
	default:
		return false
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
	if e.dict() == nil {
		return false
	}
	r, printable := k.Printable()
	if e.code != "" {
		if i, ok := k.Digit(); ok && !e.key(r) {
			if e.list.Pick(i) {
				e.choose(out)
			}
			return true
		}
		if e.navigate(k, out) {
			return true
		}
		switch k.Sym {
		case engine.KeySpace:
			e.choose(out)
			return true
		case engine.KeyReturn, engine.KeyKPEnter:
			code := e.code
			e.clear(out)
			out.Commit(code)
			return true
		case engine.KeyEscape:
			e.clear(out)
			return true
		case engine.KeyBackSpace:
			e.code = e.code[:len(e.code)-1]
			e.refresh()
			e.show(out)
			return true
		}
	}
	if !printable {
		e.Flush(out)
		return false
	}
	if k.State&engine.StateShift != 0 && r >= 'A' && r <= 'Z' {
		e.Flush(out)
		return false
	}
	if e.key(r) {
		if e.code != "" && !e.extends(r) {
			e.list.Selected = 0
			e.choose(out)
		}
		e.code += string(r)
		e.refresh()
		if e.auto && len([]rune(e.code)) == e.length && len(e.list.Items) == 1 {
			e.choose(out)
			return true
		}
		e.show(out)
		return true
	}
	e.Flush(out)
	if p, ok := (punct.Style{Language: e.Language(), FullWidth: e.opts.FullWidth}).Map(r); ok {
		out.Commit(p)
		return true
	}
	return false
}

func (e *Engine) extends(r rune) bool {
	next := e.code + string(r)
	if len([]rune(next)) > e.length {
		return false
	}
	if e.wildcard != 0 && strings.ContainsRune(next, e.wildcard) {
		return true
	}
	return e.dict().HasPrefix(next) || e.user.HasPrefix(next)
}

func (e *Engine) Select(index int, out engine.Output) {
	if e.list.Pick(index) {
		e.choose(out)
	}
}

func (e *Engine) Page(next bool, out engine.Output) {
	e.list.Turn(next)
	e.show(out)
}

func (e *Engine) Reset(out engine.Output) {
	e.clear(out)
}

func (e *Engine) Flush(out engine.Output) {
	if e.code == "" {
		return
	}
	if e.list.Empty() {
		e.clear(out)
		return
	}
	e.list.Selected = 0
	e.choose(out)
}
