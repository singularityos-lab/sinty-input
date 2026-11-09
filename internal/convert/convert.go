package convert

import (
	"strings"

	"github.com/singularityos-lab/sinty-input/internal/dict"
)

const (
	None        = ""
	Simplified  = "simplified"
	Traditional = "traditional"
	Taiwan      = "taiwan"
	HongKong    = "hongkong"
)

var Modes = []string{None, Simplified, Traditional, Taiwan, HongKong}

func IsTraditional(language string) bool {
	return language == "zh-Hant" || language == "yue" || strings.HasPrefix(language, "zh-TW") || strings.HasPrefix(language, "zh-HK")
}

func Chain(language, mode string) []string {
	if !strings.HasPrefix(language, "zh") && language != "yue" {
		return nil
	}
	if IsTraditional(language) {
		switch mode {
		case Simplified:
			return []string{"zh-t2s"}
		case Taiwan:
			return []string{"zh-tw-variants"}
		case HongKong:
			return []string{"zh-hk-variants"}
		}
		return nil
	}
	switch mode {
	case Traditional:
		return []string{"zh-s2t"}
	case Taiwan:
		return []string{"zh-s2t", "zh-tw-phrases", "zh-tw-variants"}
	case HongKong:
		return []string{"zh-s2t", "zh-hk-variants"}
	}
	return nil
}

func Apply(text string, chain []string) string {
	for _, name := range chain {
		text = Pass(text, dict.Get(name))
	}
	return text
}

func Pass(text string, d *dict.Dict) string {
	if d == nil || text == "" {
		return text
	}
	runes := []rune(text)
	longest := d.Longest()
	var b strings.Builder
	for i := 0; i < len(runes); {
		matched := false
		for n := min(longest, len(runes)-i); n >= 1; n-- {
			if list := d.Lookup(string(runes[i : i+n])); len(list) > 0 {
				b.WriteString(list[0].Text)
				i += n
				matched = true
				break
			}
		}
		if !matched {
			b.WriteRune(runes[i])
			i++
		}
	}
	return b.String()
}
