package phonetic_test

import (
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
)

func has(list []string, s string) bool {
	for _, c := range list {
		if c == s {
			return true
		}
	}
	return false
}

func TestKanaRomaji(t *testing.T) {
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
		_, r := typed(t, "kana", c.keys, engine.KeyReturn)
		if got := r.Committed.String(); got != c.want {
			t.Errorf("%q: got %q, want %q", c.keys, got, c.want)
		}
	}
}

func TestKanaCandidates(t *testing.T) {
	cases := []struct{ keys, want string }{
		{"nihon", "日本"},
		{"kanji", "漢字"},
		{"kaku", "書く"},
		{"kaita", "書いた"},
		{"tabemasu", "食べます"},
		{"konpyu-ta-", "コンピューター"},
		{"nihon", "にほん"},
	}
	for _, c := range cases {
		_, r := typed(t, "kana", c.keys, 0)
		if r.List == nil || !has(r.List.Items, c.want) {
			t.Errorf("%q: %q not offered in %+v", c.keys, c.want, r.List)
		}
	}
}

func TestKanaPreeditAndPick(t *testing.T) {
	e, r := typed(t, "kana", "nihon", 0)
	if r.Text != "にほん" {
		t.Fatalf("preedit %q", r.Text)
	}
	enginetest.Press(e, r, engine.KeySpace)
	if r.Committed.String() != "日本" || r.Text != "" {
		t.Fatalf("committed %q preedit %q", r.Committed.String(), r.Text)
	}
	enginetest.Type(e, r, "nihon")
	enginetest.Press(e, r, engine.KeyEscape)
	if r.Committed.String() != "日本" || r.Text != "" || r.List != nil {
		t.Fatalf("escape left %q", r.Text)
	}
}

func TestKanaLearnsChoice(t *testing.T) {
	e, r := typed(t, "kana", "nihon", 0)
	second := r.List.Items[1]
	enginetest.Press(e, r, '2')
	if r.Committed.String() != second {
		t.Fatalf("committed %q, want %q", r.Committed.String(), second)
	}
	enginetest.Type(e, r, "nihon")
	if r.List.Items[0] != second {
		t.Fatalf("learned choice not first: %v", r.List.Items[:3])
	}
}

func TestKanaPassesThroughWhenIdle(t *testing.T) {
	e := scheme(t, "kana")
	r := &enginetest.Recorder{}
	for _, sym := range []uint32{engine.KeySpace, engine.KeyReturn, engine.KeyBackSpace, '1'} {
		if enginetest.Press(e, r, sym) {
			t.Errorf("key %#x must reach the application", sym)
		}
	}
}
