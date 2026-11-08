package telex

import (
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
)

func typed(keys string) string {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, keys)
	e.Flush(r)
	return r.Committed.String()
}

func TestWords(t *testing.T) {
	cases := []struct{ keys, want string }{
		{"Vieetj", "Việt"},
		{"Nam", "Nam"},
		{"tieengs", "tiếng"},
		{"nguwowif", "người"},
		{"dduwowngf", "đường"},
		{"hoaf", "hoà"},
		{"khoer", "khoẻ"},
		{"thuys", "thuý"},
		{"muas", "múa"},
		{"quas", "quá"},
		{"gias", "giá"},
		{"giuwax", "giữa"},
		{"chaof", "chào"},
		{"banj", "bạn"},
		{"cuar", "của"},
		{"Aw", "Ă"},
		{"w", "ư"},
		{"tuwj", "tự"},
		{"hoawjc", "hoặc"},
		{"ass", "as"},
		{"aaa", "aa"},
		{"ddd", "dd"},
		{"asz", "a"},
		{"s", "s"},
		{"DDaaf", "Đầ"},
	}
	for _, c := range cases {
		if got := typed(c.keys); got != c.want {
			t.Errorf("%q: got %q, want %q", c.keys, got, c.want)
		}
	}
}

func TestBackSpaceRemovesLastLetter(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, "nguwowif")
	steps := []string{"ngườ", "ngư", "ng", "n", ""}
	for _, want := range steps {
		if !enginetest.Press(e, r, engine.KeyBackSpace) || r.Text != want {
			t.Fatalf("got %q, want %q", r.Text, want)
		}
	}
	if enginetest.Press(e, r, engine.KeyBackSpace) {
		t.Fatal("backspace on an empty word must reach the application")
	}
}

func TestSpaceCommitsAndPasses(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	handled := enginetest.Type(e, r, "xin chaof")
	if handled[3] {
		t.Fatal("space must reach the application")
	}
	e.Flush(r)
	if r.Committed.String() != "xinchào" {
		t.Fatalf("committed %q", r.Committed.String())
	}
}
