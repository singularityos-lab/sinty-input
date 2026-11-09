package bijoy

import (
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
)

func typeText(e *Engine, r *enginetest.Recorder, s string) {
	for _, c := range s {
		sym := uint32(c)
		if c == '\b' {
			sym = engine.KeyBackSpace
		}
		if !e.Key(engine.Key{Sym: sym}, r) {
			r.Committed.WriteRune(c)
		}
	}
	e.Flush(r)
}

func TestWords(t *testing.T) {
	cases := map[string]string{
		"jf":      "কা",
		"dj":      "কি",
		"cj":      "কে",
		"cjf":     "কো",
		"cjX":     "কৌ",
		"jgk":     "ক্ত",
		"dkgk":    "ত্তি",
		"jA":      "র্ক",
		"djA":     "র্কি",
		"jz":      "ক্র",
		"jZ":      "ক্য",
		"gf":      "আ",
		"Ffm":     "অাম",
		"fmf":     "ামা",
		"mfgdQ":   "মাইং",
		"ffgd":    "াাই",
		"evsjf":   "ডরুকা",
		"jfg,":    "কাৰ",
		"jgg":     "ক্‌",
		"123":     "১২৩",
		"Lvsg":    "ধরু্",
		"jgd":     "কই",
		"bvj ":    "নরক ",
		"jg\bk":   "কত",
		"dj\b\bk": "ত",
		"vfMNff":  "রাশষাা",
	}
	for in, want := range cases {
		e, r := New(), &enginetest.Recorder{}
		typeText(e, r, in)
		if got := r.Committed.String(); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestShortcuts(t *testing.T) {
	e, r := New(), &enginetest.Recorder{}
	if e.Key(engine.Key{Sym: 'c', State: engine.StateControl}, r) {
		t.Fatal("took Ctrl+C")
	}
}
