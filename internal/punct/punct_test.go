package punct

import "testing"

func TestStyles(t *testing.T) {
	cases := []struct {
		style Style
		in    rune
		want  string
		ok    bool
	}{
		{Style{"zh-Hans", true}, ',', "，", true},
		{Style{"zh-Hans", true}, '[', "【", true},
		{Style{"zh-Hant", true}, '[', "「", true},
		{Style{"yue", true}, '[', "「", true},
		{Style{"ja", true}, ',', "、", true},
		{Style{"ja", true}, '/', "・", true},
		{Style{"zh-Hans", true}, '#', "＃", true},
		{Style{"zh-Hans", true}, 'a', "", false},
		{Style{"zh-Hans", true}, '5', "", false},
		{Style{"zh-Hans", false}, ',', "", false},
		{Style{"en", true}, ',', "", false},
	}
	for _, c := range cases {
		got, ok := c.style.Map(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("%v %q: %q %v, want %q %v", c.style, c.in, got, ok, c.want, c.ok)
		}
	}
	if Width('A') != "Ａ" || Width(' ') != "　" {
		t.Fatal("full width")
	}
}
