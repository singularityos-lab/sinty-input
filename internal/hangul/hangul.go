package hangul

import "github.com/singularityos-lab/sinty-input/internal/engine"

var keys = map[rune]rune{
	'r': 'ㄱ', 'R': 'ㄲ', 's': 'ㄴ', 'e': 'ㄷ', 'E': 'ㄸ', 'f': 'ㄹ', 'a': 'ㅁ',
	'q': 'ㅂ', 'Q': 'ㅃ', 't': 'ㅅ', 'T': 'ㅆ', 'd': 'ㅇ', 'w': 'ㅈ', 'W': 'ㅉ',
	'c': 'ㅊ', 'z': 'ㅋ', 'x': 'ㅌ', 'v': 'ㅍ', 'g': 'ㅎ',
	'k': 'ㅏ', 'o': 'ㅐ', 'O': 'ㅒ', 'i': 'ㅑ', 'j': 'ㅓ', 'p': 'ㅔ', 'P': 'ㅖ',
	'u': 'ㅕ', 'h': 'ㅗ', 'y': 'ㅛ', 'n': 'ㅜ', 'b': 'ㅠ', 'm': 'ㅡ', 'l': 'ㅣ',
}

const (
	initials = "ㄱㄲㄴㄷㄸㄹㅁㅂㅃㅅㅆㅇㅈㅉㅊㅋㅌㅍㅎ"
	medials  = "ㅏㅐㅑㅒㅓㅔㅕㅖㅗㅘㅙㅚㅛㅜㅝㅞㅟㅠㅡㅢㅣ"
	finals   = "ㄱㄲㄳㄴㄵㄶㄷㄹㄺㄻㄼㄽㄾㄿㅀㅁㅂㅄㅅㅆㅇㅈㅊㅋㅌㅍㅎ"
)

var vowelPairs = map[[2]rune]rune{
	{'ㅗ', 'ㅏ'}: 'ㅘ', {'ㅗ', 'ㅐ'}: 'ㅙ', {'ㅗ', 'ㅣ'}: 'ㅚ',
	{'ㅜ', 'ㅓ'}: 'ㅝ', {'ㅜ', 'ㅔ'}: 'ㅞ', {'ㅜ', 'ㅣ'}: 'ㅟ',
	{'ㅡ', 'ㅣ'}: 'ㅢ',
}

var finalPairs = map[[2]rune]rune{
	{'ㄱ', 'ㅅ'}: 'ㄳ', {'ㄴ', 'ㅈ'}: 'ㄵ', {'ㄴ', 'ㅎ'}: 'ㄶ',
	{'ㄹ', 'ㄱ'}: 'ㄺ', {'ㄹ', 'ㅁ'}: 'ㄻ', {'ㄹ', 'ㅂ'}: 'ㄼ', {'ㄹ', 'ㅅ'}: 'ㄽ',
	{'ㄹ', 'ㅌ'}: 'ㄾ', {'ㄹ', 'ㅍ'}: 'ㄿ', {'ㄹ', 'ㅎ'}: 'ㅀ',
	{'ㅂ', 'ㅅ'}: 'ㅄ',
}

func index(set string, r rune) int {
	i := 0
	for _, c := range set {
		if c == r {
			return i
		}
		i++
	}
	return -1
}

func vowel(r rune) bool {
	return r >= 'ㅏ' && r <= 'ㅣ'
}

type Engine struct {
	initial rune
	vowel   []rune
	final   []rune
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Info() engine.Info {
	return engine.Info{Name: "hangul", Label: "Korean", Symbol: "한", Language: "ko"}
}

func (e *Engine) empty() bool {
	return e.initial == 0 && len(e.vowel) == 0 && len(e.final) == 0
}

func join(parts []rune, pairs map[[2]rune]rune) rune {
	switch len(parts) {
	case 0:
		return 0
	case 1:
		return parts[0]
	}
	return pairs[[2]rune{parts[0], parts[1]}]
}

func (e *Engine) text() string {
	v := join(e.vowel, vowelPairs)
	t := join(e.final, finalPairs)
	switch {
	case e.initial != 0 && v != 0:
		s := 0xac00 + (index(initials, e.initial)*21+index(medials, v))*28
		if t != 0 {
			s += index(finals, t) + 1
		}
		return string(rune(s))
	case e.initial != 0:
		return string(e.initial)
	case v != 0:
		return string(v)
	}
	return ""
}

func (e *Engine) show(out engine.Output) {
	t := e.text()
	out.Preedit(t, engine.Runes(t))
}

func (e *Engine) commit(out engine.Output) {
	if t := e.text(); t != "" {
		out.Commit(t)
	}
	e.initial, e.vowel, e.final = 0, nil, nil
}

func (e *Engine) consonant(c rune, out engine.Output) {
	switch {
	case e.initial != 0 && len(e.vowel) > 0 && len(e.final) == 0 && index(finals, c) >= 0:
		e.final = []rune{c}
	case len(e.final) == 1 && finalPairs[[2]rune{e.final[0], c}] != 0:
		e.final = append(e.final, c)
	default:
		e.commit(out)
		e.initial = c
	}
}

func (e *Engine) addVowel(v rune, out engine.Output) {
	switch {
	case len(e.final) > 0:
		next := e.final[len(e.final)-1]
		e.final = e.final[:len(e.final)-1]
		e.commit(out)
		e.initial = next
		e.vowel = []rune{v}
	case len(e.vowel) == 1 && vowelPairs[[2]rune{e.vowel[0], v}] != 0:
		e.vowel = append(e.vowel, v)
	case len(e.vowel) == 0:
		e.vowel = []rune{v}
	default:
		e.commit(out)
		e.vowel = []rune{v}
	}
}

func lookup(k engine.Key) (rune, bool) {
	r, ok := k.Printable()
	if !ok {
		return 0, false
	}
	if j, ok := keys[r]; ok {
		return j, true
	}
	if r >= 'A' && r <= 'Z' {
		j, ok := keys[r+'a'-'A']
		return j, ok
	}
	return 0, false
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
		switch {
		case len(e.final) > 0:
			e.final = e.final[:len(e.final)-1]
		case len(e.vowel) > 0:
			e.vowel = e.vowel[:len(e.vowel)-1]
		case e.initial != 0:
			e.initial = 0
		default:
			return false
		}
		e.show(out)
		return true
	}
	j, ok := lookup(k)
	if !ok {
		e.Flush(out)
		return false
	}
	if vowel(j) {
		e.addVowel(j, out)
	} else {
		e.consonant(j, out)
	}
	e.show(out)
	return true
}

func (e *Engine) Select(int, engine.Output) {}

func (e *Engine) Page(bool, engine.Output) {}

func (e *Engine) Reset(out engine.Output) {
	e.initial, e.vowel, e.final = 0, nil, nil
	out.Preedit("", 0)
}

func (e *Engine) Flush(out engine.Output) {
	if e.empty() {
		return
	}
	e.commit(out)
	out.Preedit("", 0)
}
