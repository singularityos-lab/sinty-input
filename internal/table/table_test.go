package table

import (
	"os"
	"testing"

	_ "github.com/singularityos-lab/sinty-input/internal/dict/dicttest"

	"github.com/singularityos-lab/sinty-input/internal/convert"
	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

func fresh(t *testing.T, dict string) (*Engine, *enginetest.Recorder) {
	t.Setenv("SINTY_INPUT_USER_DIR", t.TempDir())
	userdict.Forget()
	t.Cleanup(userdict.Forget)
	e := New(Scheme{Name: dict, Dict: dict})
	if !e.Available() {
		t.Fatalf("%s is not available", dict)
	}
	return e, &enginetest.Recorder{}
}

func TestWubi(t *testing.T) {
	e, r := fresh(t, "zh-wubi86")
	enginetest.Type(e, r, "wq")
	if r.List == nil || r.List.Items[0] != "你" {
		t.Fatalf("wq: %+v", r.List)
	}
	enginetest.Press(e, r, engine.KeySpace)
	if r.Committed.String() != "你" || r.Text != "" {
		t.Fatalf("committed %q preedit %q", r.Committed.String(), r.Text)
	}
	r.Committed.Reset()
	enginetest.Type(e, r, "wqvb")
	if r.List.Items[0] != "你好" {
		t.Fatalf("wqvb: %v", r.List.Items)
	}
	enginetest.Type(e, r, "g")
	if r.Committed.String() != "你好" || r.Text != "g" {
		t.Fatalf("a full code commits on the next key: %q, preedit %q", r.Committed.String(), r.Text)
	}
	enginetest.Type(e, r, ",")
	if r.Committed.String() != "你好一，" {
		t.Fatalf("punctuation commits first: %q", r.Committed.String())
	}
}

func TestAutoCommitAndWildcard(t *testing.T) {
	e, r := fresh(t, "zh-wubi86")
	enginetest.Type(e, r, "trnt")
	if r.Committed.String() != "" {
		t.Fatalf("trnt has two candidates and must wait: %q", r.Committed.String())
	}
	enginetest.Press(e, r, engine.KeyEscape)
	enginetest.Type(e, r, "wqiy")
	if r.List == nil {
		t.Fatal("wqiy shows its candidates")
	}
	enginetest.Press(e, r, engine.KeyEscape)
	var unique string
	for _, c := range e.dict().Codes() {
		if len(c) == 4 && len(e.dict().Lookup(c)) == 1 && !e.dict().HasPrefix(c+"a") {
			unique = c
			break
		}
	}
	enginetest.Type(e, r, unique)
	if r.Committed.String() != e.dict().Lookup(unique)[0].Text || r.Text != "" {
		t.Fatalf("%s should commit at once: %q", unique, r.Committed.String())
	}
	r.Committed.Reset()
	enginetest.Type(e, r, "wzvb")
	if r.List == nil || !contains(r.List.Items, "你好") {
		t.Fatalf("z matches any key: %+v", r.List)
	}
}

func TestCangjie(t *testing.T) {
	e, r := fresh(t, "zh-cangjie5")
	enginetest.Type(e, r, "hqi")
	if r.Text != "竹手戈" || r.List.Items[0] != "我" {
		t.Fatalf("hqi: %q %v", r.Text, r.List.Items)
	}
	enginetest.Press(e, r, '1')
	if r.Committed.String() != "我" {
		t.Fatalf("digit picks: %q", r.Committed.String())
	}
	r.Committed.Reset()
	e.Configure(engine.Options{Conversion: convert.Simplified, FullWidth: true, Learn: true})
	enginetest.Type(e, r, "onf ")
	enginetest.Type(e, r, "a")
	enginetest.Press(e, r, engine.KeyReturn)
	if r.Committed.String() != "你a" {
		t.Fatalf("Enter types the keys: %q", r.Committed.String())
	}
	r.Committed.Reset()
	enginetest.Type(e, r, "yrcru")
	enginetest.Press(e, r, engine.KeySpace)
	if r.Committed.String() != "说" {
		t.Fatalf("converted to simplified: %q", r.Committed.String())
	}
}

func TestLearning(t *testing.T) {
	e, r := fresh(t, "zh-quick")
	enginetest.Type(e, r, "hi")
	second := r.List.Items[1]
	enginetest.Press(e, r, '2')
	enginetest.Type(e, r, "hi")
	if r.List.Items[0] != second {
		t.Fatalf("the chosen character comes first next time: %v", r.List.Items[:3])
	}
	e.Configure(engine.Options{Learn: false})
	enginetest.Press(e, r, '3')
	userdict.FlushAll()
	if n := len(userdict.Open("zh-quick").Entries()); n != 1 {
		t.Fatalf("learning off records nothing: %d", n)
	}
	if _, err := os.Stat(userdict.Open("zh-quick").Path()); err != nil {
		t.Fatal(err)
	}
}

func TestOtherTables(t *testing.T) {
	cases := []struct{ dict, keys, want string }{
		{"zh-wubi98", "wq", "你"},
		{"zh-wubi-ng", "wq", "你"},
		{"zh-cangjie3", "hqi", "我"},
		{"zh-quick", "ab", "明"},
		{"zh-array30", "cpu", "溫"},
		{"zh-zhengma", "nr", "你"},
	}
	for _, c := range cases {
		e, r := fresh(t, c.dict)
		enginetest.Type(e, r, c.keys)
		enginetest.Press(e, r, engine.KeySpace)
		if r.Committed.String() != c.want {
			t.Errorf("%s %s: %q, want %q", c.dict, c.keys, r.Committed.String(), c.want)
		}
	}
}

func TestPassThrough(t *testing.T) {
	e, r := fresh(t, "zh-wubi86")
	if e.Key(engine.Key{Sym: 'c', State: engine.StateControl}, r) {
		t.Fatal("shortcuts belong to the application")
	}
	if e.Key(engine.Key{Sym: 'A', State: engine.StateShift}, r) {
		t.Fatal("capitals pass through")
	}
	if e.Key(engine.Key{Sym: engine.KeyBackSpace}, r) {
		t.Fatal("backspace with nothing typed belongs to the application")
	}
	if e.Key(engine.Key{Sym: '5'}, r) {
		t.Fatal("digits with nothing typed belong to the application")
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
