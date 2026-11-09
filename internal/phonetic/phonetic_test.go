package phonetic_test

import (
	"reflect"
	"testing"

	_ "github.com/singularityos-lab/sinty-input/internal/dict/dicttest"

	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/engine/enginetest"
	"github.com/singularityos-lab/sinty-input/internal/phonetic"
	"github.com/singularityos-lab/sinty-input/internal/registry"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

func scheme(t *testing.T, name string) *phonetic.Engine {
	t.Helper()
	t.Setenv("SINTY_INPUT_USER_DIR", t.TempDir())
	userdict.Forget()
	t.Cleanup(userdict.Forget)
	for _, s := range registry.Phonetic() {
		if s.Name == name {
			return phonetic.New(s)
		}
	}
	t.Fatalf("no scheme %s", name)
	return nil
}

func typed(t *testing.T, name, keys string, finish uint32) (*phonetic.Engine, *enginetest.Recorder) {
	e := scheme(t, name)
	r := &enginetest.Recorder{}
	enginetest.Type(e, r, keys)
	if finish != 0 {
		enginetest.Press(e, r, finish)
	}
	return e, r
}

func TestSegment(t *testing.T) {
	a := phonetic.NewAlphabetic(phonetic.Syllables(dict.Get("zh-pinyin")))
	cases := []struct {
		input    string
		want     []string
		ends     []int
		complete bool
	}{
		{"nihao", []string{"ni", "hao"}, []int{2, 5}, true},
		{"zhongguo", []string{"zhong", "guo"}, []int{5, 8}, true},
		{"xian", []string{"xian"}, []int{4}, true},
		{"xi'an", []string{"xi", "an"}, []int{2, 5}, true},
		{"nih", []string{"ni", "h"}, []int{2, 3}, false},
		{"woaini", []string{"wo", "ai", "ni"}, []int{2, 4, 6}, true},
	}
	for _, c := range cases {
		got, ends, complete := a.Segment(c.input)
		if !reflect.DeepEqual(got, c.want) || !reflect.DeepEqual(ends, c.ends) || complete != c.complete {
			t.Errorf("%q: got %v %v %v", c.input, got, ends, complete)
		}
	}
}

func TestPinyinPhrases(t *testing.T) {
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
		_, r := typed(t, "pinyin", c.keys, engine.KeySpace)
		if got := r.Committed.String(); got != c.want {
			t.Errorf("%q: got %q (preedit %q)", c.keys, got, r.Text)
		}
	}
}

func TestPartialChoiceKeepsRest(t *testing.T) {
	e, r := typed(t, "pinyin", "nihao", 0)
	index := -1
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

func TestEnterAndPunctuation(t *testing.T) {
	_, r := typed(t, "pinyin", "nihao", engine.KeyReturn)
	if r.Committed.String() != "nihao" {
		t.Fatalf("committed %q", r.Committed.String())
	}
	_, r = typed(t, "pinyin", "nihao,", 0)
	if r.Committed.String() != "你好，" {
		t.Fatalf("committed %q", r.Committed.String())
	}
}

func TestEmptyKeysPassThrough(t *testing.T) {
	for _, name := range []string{"pinyin", "shuangpin-xiaohe", "zhuyin", "jyutping"} {
		e := scheme(t, name)
		r := &enginetest.Recorder{}
		for _, sym := range []uint32{engine.KeySpace, engine.KeyReturn, engine.KeyBackSpace, 'A'} {
			if enginetest.Press(e, r, sym) {
				t.Errorf("%s: key %#x must reach the application", name, sym)
			}
		}
	}
}

func TestLearnsSentences(t *testing.T) {
	e, r := typed(t, "pinyin", "nihao", 0)
	var ni, hao int = -1, -1
	for i, w := range r.List.Items {
		if w == "拟" {
			ni = i
		}
	}
	if ni < 0 {
		t.Skip("no 拟 for ni")
	}
	e.Select(ni, r)
	for i, w := range r.List.Items {
		if w == "号" {
			hao = i
		}
	}
	e.Select(hao, r)
	if r.Committed.String() != "拟号" {
		t.Fatalf("committed %q", r.Committed.String())
	}
	r.Committed.Reset()
	enginetest.Type(e, r, "nihao")
	if r.List.Items[0] != "拟号" {
		t.Fatalf("the new phrase comes first: %v", r.List.Items[:3])
	}
	enginetest.Press(e, r, engine.KeySpace)
	r.Committed.Reset()
	enginetest.Type(e, r, "nihaoma")
	enginetest.Press(e, r, engine.KeySpace)
	if r.Committed.String() != "拟号吗" {
		t.Fatalf("learned phrases join sentences: %q", r.Committed.String())
	}
}

func TestShuangpinLayouts(t *testing.T) {
	syl := phonetic.Syllables(dict.Get("zh-pinyin"))
	expect := map[string]map[string]string{
		"xiaohe":    {"nihc": "ni hao", "lo": "luo", "vsgo": "zhong guo", "aa": "a", "ah": "ang", "ai": "ai", "uu": "shu", "lv": "lv", "jv": "ju", "xt": "xue"},
		"ziranma":   {"nihk": "ni hao", "vsgo": "zhong guo", "aa": "a", "ah": "ang", "ul": "shai", "jy": "jing"},
		"microsoft": {"nihk": "ni hao", "vsgo": "zhong guo", "oa": "a", "oh": "ang", "j;": "jing", "ly": "lv", "lv": "lve", "or": "er"},
		"sogou":     {"nihk": "ni hao", "vsgo": "zhong guo", "oa": "a", "lt": "lve", "j;": "jing", "ly": "lv"},
		"abc":       {"nihk": "ni hao", "asgo": "zhong guo", "oa": "a", "ec": "chuai", "vr": "shiu"},
		"ziguang":   {"nihq": "ni hao", "uhgo": "zhong guo", "oa": "a", "j;": "jing", "ap": "chai"},
	}
	for _, l := range phonetic.Layouts {
		s := phonetic.NewShuangpin(l, syl)
		seen := map[string]string{}
		for code, p := range s.Table() {
			if len(code) != 2 {
				t.Errorf("%s: code %q", l.Name, code)
			}
			seen[p] = code
		}
		for p := range syl {
			if _, ok := seen[p]; !ok && len(p) > 1 && p != "ng" && p != "hm" && p != "hng" && p != "xx" && p != "lo" && p != "yo" {
				initial, _ := phonetic.SplitPinyin(p)
				if initial != "" || p[0] == 'a' || p[0] == 'e' || p[0] == 'o' {
					t.Errorf("%s: %s has no code", l.Name, p)
				}
			}
		}
		for keys, want := range expect[l.Name] {
			segs, _, complete := s.Segment(keys)
			got := ""
			for i, x := range segs {
				if i > 0 {
					got += " "
				}
				got += x
			}
			if !syl[want] && len(segs) == 1 {
				continue
			}
			if got != want || !complete {
				t.Errorf("%s %q: %q %v, want %q", l.Name, keys, got, complete, want)
			}
		}
	}
	_, r := typed(t, "shuangpin-xiaohe", "nihc", engine.KeySpace)
	if r.Committed.String() != "你好" {
		t.Fatalf("xiaohe nihc: %q", r.Committed.String())
	}
	_, r = typed(t, "shuangpin-microsoft", "j;", 0)
	if r.List == nil || r.Text != "jing" {
		t.Fatalf("microsoft j; shows jing: %q", r.Text)
	}
}

func TestZhuyin(t *testing.T) {
	cases := map[string]string{
		"ㄓ": "zhi", "ㄓㄨㄥ": "zhong", "ㄍㄨㄛ": "guo", "ㄋㄧˇ": "", "ㄋㄧ": "ni", "ㄏㄠ": "hao",
		"ㄐㄩ": "ju", "ㄌㄩ": "lv", "ㄩㄝ": "yue", "ㄨㄥ": "weng", "ㄧㄡ": "you", "ㄉㄧㄡ": "diu", "ㄦ": "er", "ㄅㄛ": "bo",
	}
	for z, want := range cases {
		if want == "" {
			continue
		}
		if got := phonetic.ZhuyinToPinyin(z); got != want {
			t.Errorf("%s: %q, want %q", z, got, want)
		}
	}
	e, r := typed(t, "zhuyin", "su3cl3", 0)
	if r.Text != "ㄋㄧˇㄏㄠˇ" {
		t.Fatalf("preedit %q", r.Text)
	}
	enginetest.Press(e, r, engine.KeySpace)
	if r.Committed.String() != "你好" {
		t.Fatalf("zhuyin nihao: %q", r.Committed.String())
	}
	_, r = typed(t, "zhuyin", "5j/eji", engine.KeySpace)
	if r.Committed.String() != "中國" {
		t.Fatalf("zhuyin is converted to Taiwan characters: %q", r.Committed.String())
	}
	e, r = typed(t, "zhuyin", "su3", 0)
	enginetest.Press(e, r, engine.KeyDown)
	enginetest.Press(e, r, '1')
	if r.Committed.String() == "" {
		t.Fatal("after browsing, digits pick candidates")
	}
	_, r = typed(t, "zhuyin", "su3cl3", engine.KeyReturn)
	if r.Committed.String() != "ㄋㄧˇㄏㄠˇ" {
		t.Fatalf("Enter types the symbols: %q", r.Committed.String())
	}
}

func TestJyutping(t *testing.T) {
	_, r := typed(t, "jyutping", "neihou", engine.KeySpace)
	if r.Committed.String() != "你好" {
		t.Fatalf("neihou: %q", r.Committed.String())
	}
	_, r = typed(t, "jyutping", "gwongdungwaa", engine.KeySpace)
	if r.Committed.String() != "廣東話" {
		t.Fatalf("gwongdungwaa: %q", r.Committed.String())
	}
}
