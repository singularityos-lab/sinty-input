package registry

import (
	"strings"
	"unicode/utf8"

	"github.com/singularityos-lab/sinty-input/internal/bijoy"
	"github.com/singularityos-lab/sinty-input/internal/config"
	"github.com/singularityos-lab/sinty-input/internal/convert"
	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/hangul"
	"github.com/singularityos-lab/sinty-input/internal/kana"
	"github.com/singularityos-lab/sinty-input/internal/phonetic"
	"github.com/singularityos-lab/sinty-input/internal/simlish"
	"github.com/singularityos-lab/sinty-input/internal/table"
	"github.com/singularityos-lab/sinty-input/internal/telex"
)

var shuangpinLabels = map[string]string{
	"xiaohe": "Xiaohe", "ziranma": "Ziranma", "microsoft": "Microsoft", "sogou": "Sogou", "abc": "ABC", "ziguang": "Ziguang",
}

func Phonetic() []phonetic.Scheme {
	defaults := engine.DefaultOptions()
	alpha := func(s map[string]bool) phonetic.Spelling { return phonetic.NewAlphabetic(s) }
	out := []phonetic.Scheme{
		{Name: "pinyin", Label: "Chinese (Pinyin)", Symbol: "拼", Dict: "zh-pinyin", Spelling: alpha, Defaults: defaults},
	}
	for _, l := range phonetic.Layouts {
		layout := l
		out = append(out, phonetic.Scheme{
			Name: "shuangpin-" + l.Name, Label: "Chinese (Shuangpin " + shuangpinLabels[l.Name] + ")", Symbol: "双", Dict: "zh-pinyin",
			Spelling: func(s map[string]bool) phonetic.Spelling { return phonetic.NewShuangpin(layout, s) }, Defaults: defaults,
		})
	}
	zhuyin := defaults
	zhuyin.Conversion = convert.Taiwan
	out = append(out,
		phonetic.Scheme{Name: "zhuyin", Label: "Chinese (Zhuyin)", Symbol: "注", Language: "zh-Hant", Dict: "zh-pinyin",
			Spelling: func(s map[string]bool) phonetic.Spelling { return phonetic.NewZhuyin(s) }, Defaults: zhuyin},
		phonetic.Scheme{Name: "jyutping", Label: "Cantonese (Jyutping)", Symbol: "粤", Dict: "yue-jyutping", Spelling: alpha, Defaults: defaults},
		phonetic.Scheme{Name: "kana", Label: "Japanese", Symbol: "あ", Language: "ja", Dict: "ja-kana",
			Spelling: func(map[string]bool) phonetic.Spelling { return kana.Spelling{} }, Defaults: defaults, Glued: true, Rank: kana.Rank},
	)
	return out
}

func Tables() []table.Scheme {
	return []table.Scheme{
		{Name: "wubi86", Label: "Chinese (Wubi 86)", Symbol: "五", Dict: "zh-wubi86"},
		{Name: "wubi98", Label: "Chinese (Wubi 98)", Symbol: "五", Dict: "zh-wubi98"},
		{Name: "wubi-ng", Label: "Chinese (Wubi New Generation)", Symbol: "五", Dict: "zh-wubi-ng"},
		{Name: "cangjie3", Label: "Chinese (Cangjie 3)", Symbol: "倉", Dict: "zh-cangjie3"},
		{Name: "cangjie5", Label: "Chinese (Cangjie 5)", Symbol: "倉", Dict: "zh-cangjie5"},
		{Name: "quick", Label: "Chinese (Quick)", Symbol: "速", Dict: "zh-quick"},
		{Name: "array30", Label: "Chinese (Array 30)", Symbol: "行", Dict: "zh-array30"},
		{Name: "zhengma", Label: "Chinese (Zhengma)", Symbol: "郑", Dict: "zh-zhengma"},
	}
}

type availability interface {
	Available() bool
}

func Engines() []engine.Engine {
	all := []engine.Engine{}
	for _, s := range Phonetic() {
		all = append(all, phonetic.New(s))
	}
	all = append(all, hangul.New(), telex.New(), bijoy.New(), simlish.New())
	used := map[string]bool{}
	for _, s := range Tables() {
		used[s.Dict] = true
		all = append(all, table.New(s))
	}
	for _, info := range dict.Catalog() {
		if info.Meta["type"] != "table" || used[info.Name] {
			continue
		}
		symbol, _ := utf8.DecodeRuneInString(info.Meta["name"])
		all = append(all, table.New(table.Scheme{Name: "table-" + info.Name, Label: info.Meta["name"], Symbol: string(symbol), Dict: info.Name}))
	}
	var out []engine.Engine
	for _, e := range all {
		if a, ok := e.(availability); ok && !a.Available() {
			continue
		}
		out = append(out, e)
	}
	Apply(out, config.Load())
	return out
}

func Apply(engines []engine.Engine, c config.Config) {
	for _, e := range engines {
		cfg, ok := e.(engine.Configurable)
		if !ok {
			continue
		}
		m := c.Methods[e.Info().Name]
		o := cfg.Options()
		if m.Conversion != "" {
			o.Conversion = m.Conversion
		}
		o.FullWidth = m.Wide()
		o.Learn = m.Learning()
		if m.Dictionary != "" && strings.TrimSpace(m.Dictionary) != "" {
			o.Dictionary = m.Dictionary
		}
		cfg.Configure(o)
	}
}

func New(name string) engine.Engine {
	for _, e := range Engines() {
		if e.Info().Name == name {
			return e
		}
	}
	return nil
}
