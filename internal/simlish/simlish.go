package simlish

import (
	"hash/fnv"
	"strings"
	"unicode"

	"github.com/singularityos-lab/sinty-input/internal/engine"
)

var phrases = map[string]string{
	"hello":   "sul sul",
	"hi":      "sul sul",
	"bye":     "dag dag",
	"goodbye": "dag dag",
	"yes":     "ub",
	"no":      "nib",
	"thanks":  "vadish",
	"ok":      "oobee",
	"cool":    "fredishay",
	"pizza":   "chumcha",
}

var syllables = []string{
	"sul", "dag", "nib", "ub", "bu", "fre", "za", "shoo", "wib", "neeb",
	"lerb", "gerb", "fleeb", "oo", "bee", "vad", "dish", "zib", "noo", "boo",
	"cha", "mib", "plee", "bob", "glar", "nurb", "sha", "wa", "jee", "rah",
}

func translate(word string) string {
	lower := strings.ToLower(word)
	out, ok := phrases[lower]
	if !ok {
		h := fnv.New32a()
		h.Write([]byte(lower))
		seed := h.Sum32()
		n := max(1, (len([]rune(word))+2)/3)
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteString(syllables[seed%uint32(len(syllables))])
			seed = seed*1103515245 + 12345
			seed ^= seed >> 13
		}
		out = b.String()
	}
	r := []rune(word)
	if unicode.IsUpper(r[0]) {
		o := []rune(out)
		o[0] = unicode.ToUpper(o[0])
		if len(r) > 1 && strings.ToUpper(word) == word {
			return strings.ToUpper(out)
		}
		return string(o)
	}
	return out
}

type Engine struct {
	word []rune
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Info() engine.Info {
	return engine.Info{Name: "simlish", Label: "Simlish", Symbol: "♦", Language: ""}
}

func (e *Engine) show(out engine.Output) {
	t := string(e.word)
	out.Preedit(t, engine.Runes(t))
}

func (e *Engine) Key(k engine.Key, out engine.Output) bool {
	if k.Release {
		return false
	}
	if k.Shortcut() {
		e.Flush(out)
		return false
	}
	if k.Sym == engine.KeyBackSpace && len(e.word) > 0 {
		e.word = e.word[:len(e.word)-1]
		e.show(out)
		return true
	}
	if k.Sym == engine.KeyEscape && len(e.word) > 0 {
		e.Reset(out)
		return true
	}
	r, ok := k.Printable()
	if ok && unicode.IsLetter(r) {
		e.word = append(e.word, r)
		e.show(out)
		return true
	}
	e.Flush(out)
	return false
}

func (e *Engine) Select(int, engine.Output) {}

func (e *Engine) Page(bool, engine.Output) {}

func (e *Engine) Reset(out engine.Output) {
	e.word = nil
	out.Preedit("", 0)
}

func (e *Engine) Flush(out engine.Output) {
	if len(e.word) == 0 {
		return
	}
	text := translate(string(e.word))
	e.word = nil
	out.Preedit("", 0)
	out.Commit(text)
}
