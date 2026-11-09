package phonetic

import (
	"strings"
	"sync"

	"github.com/singularityos-lab/sinty-input/internal/convert"
	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/punct"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

type Scheme struct {
	Name     string
	Label    string
	Symbol   string
	Language string
	Dict     string
	Spelling func(syllables map[string]bool) Spelling
	Defaults engine.Options
}

type choice struct {
	word string
	raw  string
	code string
}

type Engine struct {
	scheme   Scheme
	opts     engine.Options
	user     *userdict.Store
	once     sync.Once
	d        *dict.Dict
	sp       Spelling
	input    string
	chosen   []choice
	list     engine.List
	spans    []int
	segs     []string
	ends     []int
	browsing bool
}

func New(s Scheme) *Engine {
	return &Engine{scheme: s, opts: s.Defaults, user: userdict.Open(s.Name)}
}

func Syllables(d *dict.Dict) map[string]bool {
	out := map[string]bool{}
	for _, code := range d.Codes() {
		if !strings.Contains(code, " ") {
			out[code] = true
		}
	}
	return out
}

func (e *Engine) load() {
	e.once.Do(func() {
		e.d = dict.Get(e.scheme.Dict)
		e.sp = e.scheme.Spelling(Syllables(e.d))
	})
}

func (e *Engine) Info() engine.Info {
	return engine.Info{Name: e.scheme.Name, Label: e.scheme.Label, Symbol: e.scheme.Symbol, Language: e.language()}
}

func (e *Engine) language() string {
	if e.scheme.Language != "" {
		return e.scheme.Language
	}
	e.load()
	return e.d.Get("language")
}

func (e *Engine) Available() bool {
	_, ok := dict.Find(e.scheme.Dict)
	return ok
}

func (e *Engine) Configure(o engine.Options) {
	if o.Conversion == "" {
		o.Conversion = e.scheme.Defaults.Conversion
	}
	e.opts = o
}

func (e *Engine) Kind() string {
	return "phonetic"
}

func (e *Engine) Dictionary() string {
	return e.scheme.Dict
}

func (e *Engine) Options() engine.Options {
	return e.opts
}

func (e *Engine) composing() bool {
	return e.input != "" || len(e.chosen) > 0
}

func (e *Engine) prefix() string {
	var b strings.Builder
	for _, c := range e.chosen {
		b.WriteString(c.word)
	}
	return b.String()
}

func (e *Engine) words(code string) []string {
	return userdict.Rank(code, e.d.Lookup(code), e.user)
}

func (e *Engine) known(code string) bool {
	return e.d.Lookup(code) != nil || e.user.Lookup(code) != nil
}

func (e *Engine) score(code string) int {
	s := e.d.Score(code)
	if list := e.user.Lookup(code); len(list) > 0 {
		s += 40 + 10*list[0].Weight
	}
	return s
}

const (
	corpus      = 180
	unreachable = -1 << 30
)

func (e *Engine) best(segs []string) string {
	n := len(segs)
	score := make([]int, n+1)
	from := make([]int, n+1)
	for i := 1; i <= n; i++ {
		score[i] = unreachable
		for j := 0; j < i; j++ {
			key := strings.Join(segs[j:i], " ")
			if score[j] == unreachable || !e.known(key) {
				continue
			}
			if s := score[j] + e.score(key) - corpus; s > score[i] {
				score[i], from[i] = s, j
			}
		}
	}
	if score[n] == unreachable {
		return ""
	}
	var words []string
	for i := n; i > 0; i = from[i] {
		words = append([]string{e.words(strings.Join(segs[from[i]:i], " "))[0]}, words...)
	}
	return strings.Join(words, "")
}

func (e *Engine) refresh() {
	e.list.Clear()
	e.spans = nil
	e.browsing = false
	segs, ends, complete := e.sp.Segment(e.input)
	e.segs, e.ends = segs, ends
	seen := map[string]bool{}
	add := func(word string, span int) {
		if word != "" && !seen[word] {
			seen[word] = true
			e.list.Items = append(e.list.Items, word)
			e.spans = append(e.spans, span)
		}
	}
	whole := len(segs)
	if !complete {
		whole--
	}
	if whole > 1 && !e.known(strings.Join(segs[:whole], " ")) {
		add(e.best(segs[:whole]), whole)
	}
	for n := whole; n >= 1; n-- {
		for _, w := range e.words(strings.Join(segs[:n], " ")) {
			add(w, n)
		}
	}
	if whole == 0 && len(segs) == 1 {
		for _, s := range e.sp.Expand(segs[0]) {
			words := e.words(s)
			for _, w := range words[:min(2, len(words))] {
				add(w, 1)
			}
		}
	}
	e.list.Aux = e.sp.Display(e.input)
}

func (e *Engine) show(out engine.Output) {
	if !e.composing() {
		out.Preedit("", 0)
		out.Candidates(nil)
		return
	}
	t := e.prefix() + e.sp.Display(e.input)
	out.Preedit(t, engine.Runes(t))
	out.Candidates(e.list.Candidates())
}

func (e *Engine) consume(count int) string {
	if count <= 0 || count > len(e.ends) {
		return e.input
	}
	return strings.TrimLeft(e.input[e.ends[count-1]:], "'")
}

func (e *Engine) choose(out engine.Output) {
	if e.list.Empty() {
		return
	}
	word, span := e.list.Current(), e.spans[e.list.Selected]
	code := strings.Join(e.segs[:span], " ")
	rest := e.consume(span)
	e.chosen = append(e.chosen, choice{word, e.input[:len(e.input)-len(rest)], code})
	e.input = rest
	if e.input == "" {
		e.learn()
		e.commit(e.prefix(), out)
		return
	}
	e.refresh()
	e.show(out)
}

func (e *Engine) learn() {
	if !e.opts.Learn || len(e.chosen) == 0 {
		return
	}
	var codes []string
	for _, c := range e.chosen {
		codes = append(codes, c.code)
		if _, ok := e.d.Weight(c.code, c.word); ok || len(e.chosen) == 1 {
			e.user.Learn(c.code, c.word)
		}
	}
	if len(e.chosen) > 1 {
		e.user.Learn(strings.Join(codes, " "), e.prefix())
	}
}

func (e *Engine) emit(text string, out engine.Output) {
	if text != "" {
		out.Commit(convert.Apply(text, convert.Chain(e.d.Get("language"), e.opts.Conversion)))
	}
}

func (e *Engine) commit(text string, out engine.Output) {
	e.emit(text, out)
	e.input, e.chosen = "", nil
	e.list.Clear()
	e.spans, e.segs, e.ends = nil, nil, nil
	e.browsing = false
	out.Preedit("", 0)
	out.Candidates(nil)
}

func (e *Engine) undo() {
	last := e.chosen[len(e.chosen)-1]
	e.chosen = e.chosen[:len(e.chosen)-1]
	e.input = last.raw + e.input
}

func (e *Engine) navigate(k engine.Key, out engine.Output) bool {
	switch k.Sym {
	case engine.KeyDown, engine.KeyRight:
		e.list.Move(1)
	case engine.KeyUp, engine.KeyLeft:
		e.list.Move(-1)
	case engine.KeyPageDown:
		e.list.Turn(true)
	case engine.KeyPageUp:
		e.list.Turn(false)
	default:
		return false
	}
	e.browsing = true
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
	e.load()
	if e.d == nil {
		return false
	}
	r, printable := k.Printable()
	if e.composing() {
		if i, ok := k.Digit(); ok && (e.browsing || !e.sp.Key(r, e.input)) {
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
			if e.list.Empty() {
				e.commit(e.prefix()+e.literal(e.input), out)
			} else {
				e.choose(out)
			}
			return true
		case engine.KeyReturn, engine.KeyKPEnter:
			var raw strings.Builder
			for _, c := range e.chosen {
				raw.WriteString(e.literal(c.raw))
			}
			text := raw.String() + e.literal(e.input)
			e.input, e.chosen = "", nil
			e.list.Clear()
			out.Preedit("", 0)
			out.Candidates(nil)
			out.Commit(text)
			return true
		case engine.KeyEscape:
			e.commit("", out)
			return true
		case engine.KeyBackSpace:
			if e.input == "" {
				e.undo()
			}
			e.input = e.input[:len(e.input)-1]
			if e.input == "" && len(e.chosen) > 0 {
				e.undo()
			}
			e.refresh()
			e.show(out)
			return true
		}
	}
	if !printable {
		e.Flush(out)
		return false
	}
	if k.State&engine.StateShift == 0 && e.sp.Key(r, e.input) {
		e.input += string(r)
		e.refresh()
		e.show(out)
		return true
	}
	e.Flush(out)
	if p, ok := (punct.Style{Language: e.language(), FullWidth: e.opts.FullWidth}).Map(r); ok {
		out.Commit(p)
		return true
	}
	return false
}

func (e *Engine) literal(input string) string {
	if _, ok := e.sp.(*Zhuyin); ok {
		return e.sp.Display(input)
	}
	return input
}

func (e *Engine) Select(index int, out engine.Output) {
	if e.list.Pick(index) {
		e.choose(out)
	}
}

func (e *Engine) Page(next bool, out engine.Output) {
	e.list.Turn(next)
	e.browsing = true
	e.show(out)
}

func (e *Engine) Reset(out engine.Output) {
	e.commit("", out)
}

func (e *Engine) Flush(out engine.Output) {
	if e.sp == nil {
		return
	}
	for e.input != "" && !e.list.Empty() {
		e.list.Selected = 0
		e.choose(out)
	}
	if e.composing() {
		e.commit(e.prefix()+e.literal(e.input), out)
	}
}
