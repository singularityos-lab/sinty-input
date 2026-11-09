package bijoy

import (
	"github.com/singularityos-lab/sinty-input/internal/engine"
)

const (
	hasanta = "্"
	zwnj    = "‌"
	reph    = "র্"
	eKar    = "ে"
)

var consonants = map[rune]string{
	'q': "ঙ", 'w': "য", 'e': "ড", 'r': "প", 't': "ট", 'y': "চ", 'u': "জ", 'i': "হ", 'o': "গ", 'p': "ড়", '\\': "ৎ",
	'h': "ব", 'j': "ক", 'k': "ত", 'l': "দ", 'v': "র", 'b': "ন", 'n': "স", 'm': "ম",
	'W': "য়", 'E': "ঢ", 'R': "ফ", 'T': "ঠ", 'Y': "ছ", 'U': "ঝ", 'I': "ঞ", 'O': "ঘ", 'P': "ঢ়",
	'H': "ভ", 'J': "খ", 'K': "থ", 'L': "ধ", 'V': "ল", 'B': "ণ", 'N': "ষ", 'M': "শ",
}

var preKars = map[rune]string{'d': "ি", 'c': "ে", 'C': "ৈ"}

var postKars = map[rune]string{'f': "া", 's': "ু", 'S': "ূ", 'a': "ৃ", 'D': "ী", 'X': "ৌ"}

var vowels = map[rune]string{
	'f': "আ", 'd': "ই", 'D': "ঈ", 's': "উ", 'S': "ঊ", 'a': "ঋ", 'c': "এ", 'C': "ঐ", 'x': "ও", 'X': "ঔ", 'G': "॥",
}

var signs = map[rune]string{
	'z': "্র", 'Z': "্য",
}

var direct = map[rune]string{
	'1': "১", '2': "২", '3': "৩", '4': "৪", '5': "৫", '6': "৬", '7': "৭", '8': "৮", '9': "৯", '0': "০",
	'x': "ও", 'F': "অ", 'G': "।", 'Q': "ং", '|': "ঃ", '&': "ঁ", '$': "৳",
	'`': "‘", '\'': "’", '~': "“", '"': "”",
}

var afterLink = map[rune]string{
	',': "ৰ", '.': "ৱ", '1': "৴", '2': "৵", '3': "৶", '4': "৷", '$': "৲", '5': "৸", '9': "ঌ", '7': "ঽ", '&': "৺",
}

type state struct {
	cluster string
	pre     string
	reph    bool
	link    bool
}

type Engine struct {
	cur  state
	undo []state
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Info() engine.Info {
	return engine.Info{Name: "bijoy", Label: "Bangla (Bijoy)", Symbol: "বা", Language: "bn"}
}

func (s state) empty() bool {
	return s.cluster == "" && s.pre == "" && !s.reph && !s.link
}

func (s state) text() string {
	t := ""
	if s.reph {
		t = reph
	}
	t += s.cluster + s.pre
	if s.link && s.cluster == "" {
		t += hasanta
	}
	return t
}

func (s state) joined() bool {
	return len(s.cluster) >= len(hasanta) && s.cluster[len(s.cluster)-len(hasanta):] == hasanta
}

func (e *Engine) show(out engine.Output) {
	t := e.cur.text()
	out.Preedit(t, engine.Runes(t))
}

func (e *Engine) finish(out engine.Output, tail string) {
	t := e.cur.text() + tail
	e.cur = state{}
	e.undo = nil
	out.Preedit("", 0)
	if t != "" {
		out.Commit(t)
	}
}

func (e *Engine) step(r rune, out engine.Output) bool {
	s := &e.cur
	if s.link && s.cluster == "" {
		s.link = false
		if v, ok := vowels[r]; ok {
			e.finish(out, v)
			return true
		}
		if v, ok := afterLink[r]; ok {
			e.finish(out, v)
			return true
		}
		if r == 'o' {
			e.finish(out, "৹")
			return true
		}
		e.finish(out, hasanta)
		return e.step(r, out)
	}
	if c, ok := consonants[r]; ok {
		switch {
		case s.joined():
			s.cluster += c
		case s.cluster != "":
			e.finish(out, "")
			e.cur.cluster = c
		default:
			s.cluster = c
		}
		e.show(out)
		return true
	}
	if r == 'g' {
		switch {
		case s.joined():
			s.cluster += zwnj
			e.finish(out, "")
		case s.cluster != "":
			s.cluster += hasanta
			e.show(out)
		default:
			if s.pre != "" {
				e.finish(out, "")
			}
			e.cur.link = true
			e.show(out)
		}
		return true
	}
	if sign, ok := signs[r]; ok {
		if s.cluster != "" && !s.joined() {
			s.cluster += sign
			e.show(out)
			return true
		}
		e.finish(out, sign)
		return true
	}
	if r == 'A' {
		if s.cluster != "" && !s.joined() {
			s.reph = true
			e.show(out)
			return true
		}
		e.finish(out, reph)
		return true
	}
	if k, ok := preKars[r]; ok {
		if s.joined() {
			s.cluster = s.cluster[:len(s.cluster)-len(hasanta)]
			e.finish(out, vowels[r])
			return true
		}
		if !s.empty() {
			e.finish(out, "")
		}
		e.cur.pre = k
		e.show(out)
		return true
	}
	if k, ok := postKars[r]; ok {
		if s.joined() {
			s.cluster = s.cluster[:len(s.cluster)-len(hasanta)]
			e.finish(out, vowels[r])
			return true
		}
		if s.cluster == "" {
			e.finish(out, k)
			return true
		}
		if s.pre == eKar && (r == 'f' || r == 'X') {
			s.pre = map[rune]string{'f': "ো", 'X': "ৌ"}[r]
			e.finish(out, "")
			return true
		}
		e.finish(out, k)
		return true
	}
	if d, ok := direct[r]; ok {
		e.finish(out, d)
		return true
	}
	e.finish(out, "")
	return false
}

func (e *Engine) Key(k engine.Key, out engine.Output) bool {
	if k.Release {
		return false
	}
	if k.Shortcut() {
		e.Flush(out)
		return false
	}
	if k.Sym == engine.KeyBackSpace && !e.cur.empty() {
		if n := len(e.undo); n > 0 {
			e.cur = e.undo[n-1]
			e.undo = e.undo[:n-1]
		} else {
			e.cur = state{}
		}
		e.show(out)
		return true
	}
	if k.Sym == engine.KeyEscape && !e.cur.empty() {
		e.Reset(out)
		return true
	}
	r, ok := k.Printable()
	if !ok || r == ' ' {
		e.Flush(out)
		return false
	}
	before := e.cur
	took := e.step(r, out)
	if !e.cur.empty() {
		e.undo = append(e.undo, before)
	}
	return took
}

func (e *Engine) Select(int, engine.Output) {}

func (e *Engine) Page(bool, engine.Output) {}

func (e *Engine) Reset(out engine.Output) {
	e.cur = state{}
	e.undo = nil
	out.Preedit("", 0)
}

func (e *Engine) Flush(out engine.Output) {
	if e.cur.empty() {
		return
	}
	e.finish(out, "")
}
