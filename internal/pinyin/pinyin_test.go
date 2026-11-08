package pinyin

import (
	"reflect"
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
)

func TestSegment(t *testing.T) {
	cases := []struct {
		input    string
		want     []string
		complete bool
	}{
		{"nihao", []string{"ni", "hao"}, true},
		{"zhongguo", []string{"zhong", "guo"}, true},
		{"xian", []string{"xian"}, true},
		{"xi'an", []string{"xi", "an"}, true},
		{"nih", []string{"ni", "h"}, false},
		{"woaini", []string{"wo", "ai", "ni"}, true},
	}
	for _, c := range cases {
		got, complete := segment(c.input)
		if !reflect.DeepEqual(got, c.want) || complete != c.complete {
			t.Errorf("%q: got %v %v", c.input, got, complete)
		}
	}
}

func typed(keys string, finish uint32) (*Engine, *enginetest.Recorder) {
	e := New()
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, keys)
	if finish != 0 {
		enginetest.Press(e, r, finish)
	}
	return e, r
}

func TestPhrases(t *testing.T) {
	cases := []struct{ keys, want string }{
		{"nihao", "你好"},
		{"zhongguo", "中国"},
		{"shi", "是"},
		{"women", "我们"},
		{"woaini", "我爱你"},
		{"jintiantianqihenhao", "今天天气很好"},
		{"wozaixuexizhongwen", "我在学习中文"},
		{"tamenqubeijing", "他们去北京"},
		{"womendewenti", "我们的问题"},
		{"womenyiqichifan", "我们一起吃饭"},
	}
	for _, c := range cases {
		_, r := typed(c.keys, engine.KeySpace)
		if got := r.Committed.String(); got != c.want {
			t.Errorf("%q: got %q (preedit %q)", c.keys, got, r.Text)
		}
	}
}

func TestPartialChoiceKeepsRest(t *testing.T) {
	e, r := typed("nihao", 0)
	var index = -1
	for i, w := range r.List.Items {
		if w == "你" {
			index = i
		}
	}
	if index < 0 {
		t.Fatalf("candidates %v", r.List.Items)
	}
	e.Select(index, r)
	if r.Committed.String() != "" || r.Text != "你hao" {
		t.Fatalf("committed %q preedit %q", r.Committed.String(), r.Text)
	}
	for range 3 {
		enginetest.Press(e, r, engine.KeyBackSpace)
	}
	if r.Text != "ni" {
		t.Fatalf("deleting the rest must undo the choice, preedit %q", r.Text)
	}
}

func TestEnterCommitsLetters(t *testing.T) {
	_, r := typed("nihao", engine.KeyReturn)
	if r.Committed.String() != "nihao" {
		t.Fatalf("committed %q", r.Committed.String())
	}
}

func TestPunctuation(t *testing.T) {
	_, r := typed("nihao,", 0)
	if r.Committed.String() != "你好，" {
		t.Fatalf("committed %q", r.Committed.String())
	}
}

func TestEmptyKeysPassThrough(t *testing.T) {
	e := New()
	r := &enginetest.Recorder{}
	for _, sym := range []uint32{engine.KeySpace, engine.KeyReturn, engine.KeyBackSpace, '1', 'A'} {
		if enginetest.Press(e, r, sym) {
			t.Errorf("key %#x must reach the application", sym)
		}
	}
}
