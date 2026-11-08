package hangul

import (
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
)

func compose(keys string) (string, string) {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, keys)
	committed := r.Committed.String()
	e.Flush(r)
	return committed, r.Committed.String()
}

func TestCompose(t *testing.T) {
	cases := []struct{ keys, want string }{
		{"gksrmf", "한글"},
		{"dkssudgktpdy", "안녕하세요"},
		{"rhkd", "광"},
		{"dmlwk", "의자"},
		{"ekfr", "닭"},
		{"ekfrdl", "닭이"},
		{"ekfrl", "달기"},
		{"rkqt", "값"},
		{"rkqtl", "갑시"},
		{"Rk", "까"},
		{"ekE", "다ㄸ"},
		{"k", "ㅏ"},
		{"hk", "ㅘ"},
		{"rr", "ㄱㄱ"},
		{"dnpdl", "웨이"},
		{"tkfka", "사람"},
	}
	for _, c := range cases {
		if _, got := compose(c.keys); got != c.want {
			t.Errorf("%q: got %q, want %q", c.keys, got, c.want)
		}
	}
}

func TestPreeditHoldsSyllable(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, "gks")
	if r.Text != "한" || r.Committed.String() != "" || r.Cursor != 1 {
		t.Fatalf("preedit %q committed %q cursor %d", r.Text, r.Committed.String(), r.Cursor)
	}
	enginetest.Type(e, r, "r")
	if r.Text != "ㄱ" || r.Committed.String() != "한" {
		t.Fatalf("preedit %q committed %q", r.Text, r.Committed.String())
	}
}

func TestBackSpace(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, "ekfr")
	steps := []string{"달", "다", "ㄷ", ""}
	for _, want := range steps {
		if !enginetest.Press(e, r, engine.KeyBackSpace) || r.Text != want {
			t.Fatalf("got %q, want %q", r.Text, want)
		}
	}
	if enginetest.Press(e, r, engine.KeyBackSpace) {
		t.Fatal("backspace on empty composition must reach the application")
	}
}

func TestForeignKeyFlushes(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	handled := enginetest.Type(e, r, "gks1")
	if handled[3] || r.Committed.String() != "한" || r.Text != "" {
		t.Fatalf("handled %v committed %q preedit %q", handled, r.Committed.String(), r.Text)
	}
}

func TestShortcutPassesThrough(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, "gk")
	if e.Key(engine.Key{Sym: 'c', State: engine.StateControl}, r) {
		t.Fatal("ctrl+c must reach the application")
	}
	if r.Committed.String() != "하" {
		t.Fatalf("committed %q", r.Committed.String())
	}
}

func TestCapsLockTypesUnshifted(t *testing.T) {
	if _, got := compose("GKS"); got != "한" {
		t.Fatalf("got %q", got)
	}
}
