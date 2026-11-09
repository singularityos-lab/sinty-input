package kana

import (
	_ "github.com/singularityos-lab/sinty-input/internal/dict/dicttest"
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
)

func TestRomaji(t *testing.T) {
	cases := []struct{ keys, want string }{
		{"konnnichiha", "こんにちは"},
		{"konnichiha", "こんいちは"},
		{"kitte", "きって"},
		{"shinbun", "しんぶん"},
		{"kan'i", "かんい"},
		{"kyouto", "きょうと"},
		{"tokyo", "ときょ"},
		{"ja-", "じゃー"},
		{"fairu", "ふぁいる"},
		{"matcha", "まっちゃ"},
		{"hon.", "ほん。"},
		{"n", "ん"},
	}
	for _, c := range cases {
		e := New()
		r := &enginetest.Recorder{}
		enginetest.Type(e, r, c.keys)
		enginetest.Press(e, r, engine.KeyReturn)
		if got := r.Committed.String(); got != c.want {
			t.Errorf("%q: got %q, want %q", c.keys, got, c.want)
		}
	}
}

func converted(keys string) (*Engine, *enginetest.Recorder) {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, keys)
	enginetest.Press(e, r, engine.KeySpace)
	return e, r
}

func contains(list []string, s string) bool {
	for _, c := range list {
		if c == s {
			return true
		}
	}
	return false
}

func TestConversion(t *testing.T) {
	cases := []struct{ keys, first string }{
		{"nihon", "日本"},
		{"kanji", "漢字"},
		{"kaku", "書く"},
		{"kaita", "書いた"},
		{"tabemasu", "食べます"},
	}
	for _, c := range cases {
		_, r := converted(c.keys)
		if r.List == nil || !contains(r.List.Items, c.first) {
			t.Errorf("%q: %q not on the first page of %+v", c.keys, c.first, r.List)
		}
	}
}

func TestCandidateCycleAndCommit(t *testing.T) {
	e, r := converted("nihon")
	enginetest.Press(e, r, engine.KeySpace)
	if r.Text != "二本" {
		t.Fatalf("second candidate %q", r.Text)
	}
	enginetest.Press(e, r, engine.KeyReturn)
	if r.Committed.String() != "二本" || r.Text != "" || r.List != nil {
		t.Fatalf("committed %q preedit %q", r.Committed.String(), r.Text)
	}
}

func TestKatakanaIsOffered(t *testing.T) {
	_, r := converted("konpyu-ta-")
	if !contains(r.List.Items, "コンピューター") {
		t.Fatalf("candidates %v", r.List.Items)
	}
}

func TestDigitSelects(t *testing.T) {
	e, r := converted("kanji")
	enginetest.Press(e, r, '2')
	if r.Committed.String() != "幹事" {
		t.Fatalf("committed %q", r.Committed.String())
	}
}

func TestEscapeReturnsToKana(t *testing.T) {
	e, r := converted("nihon")
	enginetest.Press(e, r, engine.KeyEscape)
	if r.Text != "にほん" || r.List != nil {
		t.Fatalf("preedit %q", r.Text)
	}
}

func TestTypingAfterConversionCommits(t *testing.T) {
	e, r := converted("nihon")
	enginetest.Type(e, r, "go")
	if r.Committed.String() != "日本" || r.Text != "ご" {
		t.Fatalf("committed %q preedit %q", r.Committed.String(), r.Text)
	}
}

func TestEmptyKeysPassThrough(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	for _, sym := range []uint32{engine.KeySpace, engine.KeyReturn, engine.KeyBackSpace, '1'} {
		if enginetest.Press(e, r, sym) {
			t.Errorf("key %#x must reach the application", sym)
		}
	}
}
