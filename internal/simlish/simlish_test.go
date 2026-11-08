package simlish

import (
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
)

func typeText(e *Engine, r *enginetest.Recorder, s string) {
	for _, c := range s {
		sym := uint32(c)
		switch c {
		case '\n':
			sym = engine.KeyReturn
		case '\b':
			sym = engine.KeyBackSpace
		}
		if !e.Key(engine.Key{Sym: sym}, r) {
			r.Committed.WriteRune(c)
		}
	}
}

func TestPhrases(t *testing.T) {
	e, r := New(), &enginetest.Recorder{}
	typeText(e, r, "Hello, it's there! Goodbye\n")
	if got := r.Committed.String(); got[:9] != "Sul sul, " || got[len(got)-8:] != "Dag dag\n" {
		t.Fatalf("got %q", got)
	}
}

func TestStableGibberish(t *testing.T) {
	a, b := translate("refund"), translate("refund")
	if a != b || a == "refund" || a == "" {
		t.Fatalf("%q %q", a, b)
	}
	if translate("SIMLISH")[0] < 'A' || translate("SIMLISH")[0] > 'Z' {
		t.Fatal("shouting is lost")
	}
}

func TestBackspaceAndShortcuts(t *testing.T) {
	e, r := New(), &enginetest.Recorder{}
	typeText(e, r, "hix\b ")
	if r.Committed.String() != "sul sul " {
		t.Fatalf("got %q", r.Committed.String())
	}
	if e.Key(engine.Key{Sym: 'c', State: engine.StateControl}, r) {
		t.Fatal("took Ctrl+C")
	}
}
