package telex

import (
	"strings"
	"unicode"

	"github.com/singularityos-lab/sinty-input/internal/engine"
)

var toned = map[rune]string{
	'a': "aáàảãạ", 'ă': "ăắằẳẵặ", 'â': "âấầẩẫậ",
	'e': "eéèẻẽẹ", 'ê': "êếềểễệ",
	'i': "iíìỉĩị",
	'o': "oóòỏõọ", 'ô': "ôốồổỗộ", 'ơ': "ơớờởỡợ",
	'u': "uúùủũụ", 'ư': "ưứừửữự",
	'y': "yýỳỷỹỵ",
}

var circumflex = map[rune]rune{'a': 'â', 'e': 'ê', 'o': 'ô'}

var horn = map[rune]rune{'a': 'ă', 'o': 'ơ', 'u': 'ư'}

var spelled = map[rune]string{
	'ă': "aw", 'â': "aa", 'ê': "ee", 'ô': "oo", 'ơ': "ow", 'ư': "uw", 'đ': "dd",
}

const tones = " sfrxj"

type letter struct {
	base  rune
	upper bool
}

func isVowel(r rune) bool {
	_, ok := toned[r]
	return ok
}

func plain(r rune) rune {
	for b, h := range circumflex {
		if h == r {
			return b
		}
	}
	for b, h := range horn {
		if h == r {
			return b
		}
	}
	if r == 'đ' {
		return 'd'
	}
	return r
}

type word struct {
	letters []letter
	tone    int
}

func (w *word) add(r rune, upper bool) {
	w.letters = append(w.letters, letter{r, upper})
}

func (w *word) vowels() (int, int) {
	end := -1
	for i := len(w.letters) - 1; i >= 0; i-- {
		if isVowel(w.letters[i].base) {
			end = i
			break
		}
	}
	if end < 0 {
		return -1, -1
	}
	start := end
	for start > 0 && isVowel(w.letters[start-1].base) {
		start--
	}
	return start, end + 1
}

func (w *word) circumflex(c rune, upper bool) {
	start, end := w.vowels()
	for i := end - 1; i >= start && i >= 0; i-- {
		switch w.letters[i].base {
		case c:
			w.letters[i].base = circumflex[c]
			return
		case circumflex[c]:
			w.letters[i].base = c
			w.add(c, upper)
			return
		}
	}
	w.add(c, upper)
}

func (w *word) horn(upper bool) {
	start, end := w.vowels()
	if start < 0 {
		w.add('ư', upper)
		return
	}
	for i := start; i+1 < end; i++ {
		a, b := w.letters[i].base, w.letters[i+1].base
		if a == 'u' && b == 'o' {
			w.letters[i].base, w.letters[i+1].base = 'ư', 'ơ'
			return
		}
		if a == 'ư' && b == 'ơ' {
			w.letters[i].base, w.letters[i+1].base = 'u', 'o'
			w.add('w', upper)
			return
		}
	}
	for i := end - 1; i >= start; i-- {
		b := w.letters[i].base
		if h, ok := horn[b]; ok {
			w.letters[i].base = h
			return
		}
		if p := plain(b); p != b && horn[p] == b {
			w.letters[i].base = p
			w.add('w', upper)
			return
		}
	}
	w.add('w', upper)
}

func (w *word) stroke(upper bool) {
	for i := range w.letters {
		switch w.letters[i].base {
		case 'd':
			w.letters[i].base = 'đ'
			return
		case 'đ':
			w.letters[i].base = 'd'
			w.add('d', upper)
			return
		}
	}
	w.add('d', upper)
}

func (w *word) key(c rune) {
	lc := unicode.ToLower(c)
	upper := lc != c
	t := strings.IndexRune(tones, lc)
	start, _ := w.vowels()
	switch {
	case t > 0 && start >= 0:
		if w.tone == t {
			w.tone = 0
			w.add(lc, upper)
		} else {
			w.tone = t
		}
	case lc == 'z' && w.tone != 0:
		w.tone = 0
	case circumflex[lc] != 0:
		w.circumflex(lc, upper)
	case lc == 'w':
		w.horn(upper)
	case lc == 'd':
		w.stroke(upper)
	default:
		w.add(lc, upper)
	}
}

func (w *word) toneAt() int {
	start, end := w.vowels()
	if start < 0 {
		return -1
	}
	if start > 0 {
		prev := w.letters[start-1].base
		if prev == 'q' && w.letters[start].base == 'u' && end-start > 1 {
			start++
		}
		if prev == 'g' && w.letters[start].base == 'i' && end-start > 1 {
			start++
		}
	}
	for i := end - 1; i >= start; i-- {
		if plain(w.letters[i].base) != w.letters[i].base {
			return i
		}
	}
	switch {
	case end-start == 1:
		return start
	case end < len(w.letters):
		return end - 1
	case end-start == 3:
		return start + 1
	}
	pair := string([]rune{w.letters[start].base, w.letters[start+1].base})
	if pair == "oa" || pair == "oe" || pair == "uy" {
		return start + 1
	}
	return start
}

func (w *word) String() string {
	at := -1
	if w.tone > 0 {
		at = w.toneAt()
	}
	var b strings.Builder
	for i, l := range w.letters {
		r := l.base
		if i == at {
			r = []rune(toned[r])[w.tone]
		}
		if l.upper {
			r = unicode.ToUpper(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func untone(r rune) (rune, int) {
	for base, forms := range toned {
		for i, f := range []rune(forms) {
			if f == r {
				return base, i
			}
		}
	}
	return r, 0
}

func spell(text string) []rune {
	var keys []rune
	tone := 0
	for _, r := range text {
		lr := unicode.ToLower(r)
		base, t := untone(lr)
		if t > 0 {
			tone = t
		}
		s, ok := spelled[base]
		if !ok {
			s = string(base)
		}
		for i, c := range s {
			if i == 0 && lr != r {
				c = unicode.ToUpper(c)
			}
			keys = append(keys, c)
		}
	}
	if tone > 0 {
		keys = append(keys, rune(tones[tone]))
	}
	return keys
}

type Engine struct {
	keys []rune
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Info() engine.Info {
	return engine.Info{Name: "telex", Label: "Vietnamese (Telex)", Symbol: "VI", Language: "vi"}
}

func (e *Engine) text() string {
	w := &word{}
	for _, c := range e.keys {
		w.key(c)
	}
	return w.String()
}

func (e *Engine) show(out engine.Output) {
	t := e.text()
	out.Preedit(t, engine.Runes(t))
}

func letterKey(k engine.Key) (rune, bool) {
	r, ok := k.Printable()
	if !ok || !unicode.IsLetter(r) {
		return 0, false
	}
	return r, true
}

func (e *Engine) Key(k engine.Key, out engine.Output) bool {
	if k.Release {
		return false
	}
	if k.Shortcut() {
		e.Flush(out)
		return false
	}
	if k.Sym == engine.KeyBackSpace {
		if len(e.keys) == 0 {
			return false
		}
		r := []rune(e.text())
		e.keys = spell(string(r[:len(r)-1]))
		e.show(out)
		return true
	}
	c, ok := letterKey(k)
	if !ok {
		e.Flush(out)
		return false
	}
	e.keys = append(e.keys, c)
	e.show(out)
	return true
}

func (e *Engine) Select(int, engine.Output) {}

func (e *Engine) Page(bool, engine.Output) {}

func (e *Engine) Reset(out engine.Output) {
	e.keys = nil
	out.Preedit("", 0)
}

func (e *Engine) Flush(out engine.Output) {
	if len(e.keys) == 0 {
		return
	}
	if t := e.text(); t != "" {
		out.Commit(t)
	}
	e.keys = nil
	out.Preedit("", 0)
}
