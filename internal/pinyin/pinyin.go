package pinyin

import (
	"sort"
	"strings"
	"sync"

	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/engine"
)

var (
	syllableOnce sync.Once
	syllables    map[string]bool
	partials     map[string][]string
	longest      int
)

func loadSyllables() {
	syllables, partials = map[string]bool{}, map[string][]string{}
	dict.Chinese.Each(func(key string, _ []string) {
		if !strings.Contains(key, " ") {
			syllables[key] = true
		}
	})
	for s := range syllables {
		longest = max(longest, len(s))
		for i := 1; i < len(s); i++ {
			partials[s[:i]] = append(partials[s[:i]], s)
		}
	}
	for _, list := range partials {
		sort.Strings(list)
	}
}

func segment(input string) ([]string, bool) {
	syllableOnce.Do(loadSyllables)
	var out []string
	for _, part := range strings.Split(input, "'") {
		if part == "" {
			continue
		}
		segs, ok := split(part)
		out = append(out, segs...)
		if !ok {
			return out, false
		}
	}
	return out, true
}

func split(s string) ([]string, bool) {
	n := len(s)
	best := make([]int, n+1)
	from := make([]int, n+1)
	for i := 1; i <= n; i++ {
		best[i] = -1
		for l := min(longest, i); l >= 1; l-- {
			if best[i-l] >= 0 && syllables[s[i-l:i]] && (best[i] < 0 || best[i-l]+1 < best[i]) {
				best[i], from[i] = best[i-l]+1, i-l
			}
		}
	}
	if best[n] >= 0 {
		return walk(s, from, n), true
	}
	for i := n - 1; i >= 0; i-- {
		if best[i] >= 0 && partials[s[i:]] != nil {
			return append(walk(s, from, i), s[i:]), false
		}
	}
	return []string{s}, false
}

func walk(s string, from []int, end int) []string {
	var out []string
	for end > 0 {
		out = append([]string{s[from[end]:end]}, out...)
		end = from[end]
	}
	return out
}

type choice struct {
	word  string
	input string
}

type Engine struct {
	input  string
	chosen []choice
	list   engine.List
	spans  []int
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Info() engine.Info {
	return engine.Info{Name: "pinyin", Label: "Chinese (Pinyin)", Symbol: "拼", Language: "zh"}
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

const (
	corpus      = 180
	unreachable = -1 << 30
)

func best(segs []string) string {
	n := len(segs)
	score := make([]int, n+1)
	from := make([]int, n+1)
	for i := 1; i <= n; i++ {
		score[i] = unreachable
		for j := 0; j < i; j++ {
			key := strings.Join(segs[j:i], " ")
			if score[j] == unreachable || dict.Chinese.Lookup(key) == nil {
				continue
			}
			if s := score[j] + dict.Chinese.Score(key) - corpus; s > score[i] {
				score[i], from[i] = s, j
			}
		}
	}
	if score[n] == unreachable {
		return ""
	}
	var words []string
	for i := n; i > 0; i = from[i] {
		words = append([]string{dict.Chinese.Lookup(strings.Join(segs[from[i]:i], " "))[0]}, words...)
	}
	return strings.Join(words, "")
}

func (e *Engine) refresh() {
	e.list.Clear()
	e.spans = nil
	segs, complete := segment(e.input)
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
	if whole > 1 && dict.Chinese.Lookup(strings.Join(segs[:whole], " ")) == nil {
		add(best(segs[:whole]), whole)
	}
	for n := whole; n >= 1; n-- {
		for _, w := range dict.Chinese.Lookup(strings.Join(segs[:n], " ")) {
			add(w, n)
		}
	}
	if whole == 0 && len(segs) == 1 {
		for _, s := range partials[segs[0]] {
			words := dict.Chinese.Lookup(s)
			for _, w := range words[:min(2, len(words))] {
				add(w, 1)
			}
		}
	}
	e.list.Aux = strings.Join(segs, "'")
}

func (e *Engine) show(out engine.Output) {
	if !e.composing() {
		out.Preedit("", 0)
		out.Candidates(nil)
		return
	}
	t := e.prefix() + e.list.Aux
	out.Preedit(t, engine.Runes(t))
	out.Candidates(e.list.Candidates())
}

func consume(input string, count int) string {
	segs, _ := segment(input)
	rest := input
	for _, s := range segs[:min(count, len(segs))] {
		rest = strings.TrimPrefix(strings.TrimLeft(rest, "'"), s)
	}
	return strings.TrimLeft(rest, "'")
}

func (e *Engine) choose(out engine.Output) {
	if e.list.Empty() {
		return
	}
	word, span := e.list.Current(), e.spans[e.list.Selected]
	rest := consume(e.input, span)
	e.chosen = append(e.chosen, choice{word, e.input[:len(e.input)-len(rest)]})
	e.input = rest
	if e.input == "" {
		e.commit(e.prefix(), out)
		return
	}
	e.refresh()
	e.show(out)
}

func (e *Engine) commit(text string, out engine.Output) {
	if text != "" {
		out.Commit(text)
	}
	e.input, e.chosen = "", nil
	e.list.Clear()
	e.spans = nil
	out.Preedit("", 0)
	out.Candidates(nil)
}

func (e *Engine) undo() {
	last := e.chosen[len(e.chosen)-1]
	e.chosen = e.chosen[:len(e.chosen)-1]
	e.input = last.input + e.input
}

var punctuation = map[rune]string{
	',': "，", '.': "。", '?': "？", '!': "！", ':': "：", ';': "；",
	'(': "（", ')': "）", '[': "【", ']': "】", '<': "《", '>': "》",
	'\\': "、", '~': "～", '$': "￥", '^': "……", '_': strings.Repeat(string(rune(0x2014)), 2),
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
	if e.composing() {
		if i, ok := k.Digit(); ok {
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
				e.commit(e.prefix()+e.input, out)
			} else {
				e.choose(out)
			}
			return true
		case engine.KeyReturn, engine.KeyKPEnter:
			var raw strings.Builder
			for _, c := range e.chosen {
				raw.WriteString(c.input)
			}
			e.commit(raw.String()+e.input, out)
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
	r, ok := k.Printable()
	if !ok {
		e.Flush(out)
		return false
	}
	if (r >= 'a' && r <= 'z') || (r == '\'' && e.input != "") {
		e.input += string(r)
		e.refresh()
		e.show(out)
		return true
	}
	if p, ok := punctuation[r]; ok {
		e.Flush(out)
		out.Commit(p)
		return true
	}
	e.Flush(out)
	return false
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
	e.commit("", out)
}

func (e *Engine) Flush(out engine.Output) {
	for e.input != "" && !e.list.Empty() {
		e.list.Selected = 0
		e.choose(out)
	}
	if e.composing() {
		e.commit(e.prefix()+e.input, out)
	}
}
