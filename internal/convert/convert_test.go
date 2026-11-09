package convert

import (
	"testing"

	_ "github.com/singularityos-lab/sinty-input/internal/dict/dicttest"
)

func TestConvert(t *testing.T) {
	cases := []struct {
		language, mode, in, want string
	}{
		{"zh-Hans", Traditional, "头发和发展", "頭髮和發展"},
		{"zh-Hans", Taiwan, "鼠标和软件", "滑鼠和軟體"},
		{"zh-Hans", HongKong, "为什么", "為什麼"},
		{"zh-Hans", Traditional, "为什么", "爲什麼"},
		{"zh-Hans", None, "汉字", "汉字"},
		{"zh-Hans", Simplified, "汉字", "汉字"},
		{"zh-Hant", Simplified, "漢字和頭髮", "汉字和头发"},
		{"zh-Hant", Taiwan, "爲", "為"},
		{"yue", Simplified, "廣東話", "广东话"},
		{"ja", Simplified, "漢字", "漢字"},
	}
	for _, c := range cases {
		if got := Apply(c.in, Chain(c.language, c.mode)); got != c.want {
			t.Errorf("%s %s %q: %q, want %q", c.language, c.mode, c.in, got, c.want)
		}
	}
}
