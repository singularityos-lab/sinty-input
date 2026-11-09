package phonetic

import (
	"strings"
)

var dachen = map[rune]rune{
	'1': 'ㄅ', 'q': 'ㄆ', 'a': 'ㄇ', 'z': 'ㄈ', '2': 'ㄉ', 'w': 'ㄊ', 's': 'ㄋ', 'x': 'ㄌ',
	'e': 'ㄍ', 'd': 'ㄎ', 'c': 'ㄏ', 'r': 'ㄐ', 'f': 'ㄑ', 'v': 'ㄒ', '5': 'ㄓ', 't': 'ㄔ',
	'g': 'ㄕ', 'b': 'ㄖ', 'y': 'ㄗ', 'h': 'ㄘ', 'n': 'ㄙ', 'u': 'ㄧ', 'j': 'ㄨ', 'm': 'ㄩ',
	'8': 'ㄚ', 'i': 'ㄛ', 'k': 'ㄜ', ',': 'ㄝ', '9': 'ㄞ', 'o': 'ㄟ', 'l': 'ㄠ', '.': 'ㄡ',
	'0': 'ㄢ', 'p': 'ㄣ', ';': 'ㄤ', '/': 'ㄥ', '-': 'ㄦ',
}

var tones = map[rune]rune{'3': 'ˇ', '4': 'ˋ', '6': 'ˊ', '7': '˙'}

var zhuyinInitials = map[rune]string{
	'ㄅ': "b", 'ㄆ': "p", 'ㄇ': "m", 'ㄈ': "f", 'ㄉ': "d", 'ㄊ': "t", 'ㄋ': "n", 'ㄌ': "l",
	'ㄍ': "g", 'ㄎ': "k", 'ㄏ': "h", 'ㄐ': "j", 'ㄑ': "q", 'ㄒ': "x", 'ㄓ': "zh", 'ㄔ': "ch",
	'ㄕ': "sh", 'ㄖ': "r", 'ㄗ': "z", 'ㄘ': "c", 'ㄙ': "s",
}

var zhuyinRimes = map[string]string{
	"ㄚ": "a", "ㄛ": "o", "ㄜ": "e", "ㄞ": "ai", "ㄟ": "ei", "ㄠ": "ao", "ㄡ": "ou",
	"ㄢ": "an", "ㄣ": "en", "ㄤ": "ang", "ㄥ": "eng", "ㄦ": "er",
	"ㄧ": "i", "ㄧㄚ": "ia", "ㄧㄛ": "io", "ㄧㄝ": "ie", "ㄧㄞ": "iai", "ㄧㄠ": "iao", "ㄧㄡ": "iu",
	"ㄧㄢ": "ian", "ㄧㄣ": "in", "ㄧㄤ": "iang", "ㄧㄥ": "ing",
	"ㄨ": "u", "ㄨㄚ": "ua", "ㄨㄛ": "uo", "ㄨㄞ": "uai", "ㄨㄟ": "ui", "ㄨㄢ": "uan", "ㄨㄣ": "un",
	"ㄨㄤ": "uang", "ㄨㄥ": "ong",
	"ㄩ": "ü", "ㄩㄝ": "üe", "ㄩㄢ": "üan", "ㄩㄣ": "ün", "ㄩㄥ": "iong",
}

var standalone = map[string]string{
	"i": "yi", "ia": "ya", "io": "yo", "ie": "ye", "iai": "yai", "iao": "yao", "iu": "you",
	"ian": "yan", "in": "yin", "iang": "yang", "ing": "ying",
	"u": "wu", "ua": "wa", "uo": "wo", "uai": "wai", "ui": "wei", "uan": "wan", "un": "wen",
	"uang": "wang", "ong": "weng",
	"ü": "yu", "üe": "yue", "üan": "yuan", "ün": "yun", "iong": "yong",
}

func ZhuyinToPinyin(z string) string {
	runes := []rune(z)
	if len(runes) == 0 {
		return ""
	}
	initial, ok := zhuyinInitials[runes[0]]
	rime := string(runes)
	if ok {
		rime = string(runes[1:])
	}
	if rime == "" {
		switch initial {
		case "zh", "ch", "sh", "r", "z", "c", "s":
			return initial + "i"
		}
		return ""
	}
	final, ok := zhuyinRimes[rime]
	if !ok {
		return ""
	}
	if initial == "" {
		if s, ok := standalone[final]; ok {
			return s
		}
		return final
	}
	switch initial {
	case "j", "q", "x":
		final = strings.ReplaceAll(final, "ü", "u")
	default:
		final = strings.ReplaceAll(final, "ü", "v")
	}
	return initial + final
}

type Zhuyin struct {
	pinyin   map[string]string
	prefixes map[string][]string
}

func NewZhuyin(syllables map[string]bool) *Zhuyin {
	z := &Zhuyin{pinyin: map[string]string{}, prefixes: map[string][]string{}}
	initials := []string{""}
	for r := range zhuyinInitials {
		initials = append(initials, string(r))
	}
	rimes := []string{""}
	for r := range zhuyinRimes {
		rimes = append(rimes, r)
	}
	for _, i := range initials {
		for _, r := range rimes {
			key := i + r
			p := ZhuyinToPinyin(key)
			if p == "" || !syllables[p] {
				continue
			}
			z.pinyin[key] = p
			runes := []rune(key)
			for n := 1; n < len(runes); n++ {
				z.prefixes[string(runes[:n])] = append(z.prefixes[string(runes[:n])], p)
			}
		}
	}
	return z
}

func (z *Zhuyin) Key(r rune, input string) bool {
	if _, ok := dachen[r]; ok {
		return true
	}
	_, ok := tones[r]
	return ok && input != ""
}

func kind(r rune) int {
	switch {
	case zhuyinInitials[r] != "":
		return 0
	case r == 'ㄧ' || r == 'ㄨ' || r == 'ㄩ':
		return 1
	default:
		return 2
	}
}

func (z *Zhuyin) Segment(input string) ([]string, []int, bool) {
	var segs []string
	var ends []int
	var cur []rune
	parts := [3]bool{}
	closeAt := func(end int) bool {
		if len(cur) == 0 {
			return true
		}
		key := string(cur)
		if p, ok := z.pinyin[key]; ok {
			segs = append(segs, p)
			ends = append(ends, end)
			cur, parts = nil, [3]bool{}
			return true
		}
		return false
	}
	for i, k := range input {
		if _, ok := tones[k]; ok {
			if !closeAt(i) {
				return append(segs, string(cur)), append(ends, len(input)), false
			}
			if len(ends) > 0 {
				ends[len(ends)-1] = i + 1
			}
			continue
		}
		sym := dachen[k]
		c := kind(sym)
		split := false
		switch c {
		case 0:
			split = len(cur) > 0
		case 1:
			split = parts[1] || parts[2]
		case 2:
			split = parts[2]
		}
		if split && !closeAt(i) {
			return append(segs, string(cur)), append(ends, len(input)), false
		}
		cur = append(cur, sym)
		parts[c] = true
	}
	if len(cur) > 0 && !closeAt(len(input)) {
		return append(segs, string(cur)), append(ends, len(input)), false
	}
	return segs, ends, true
}

func (z *Zhuyin) Expand(partial string) []string {
	return z.prefixes[partial]
}

func (z *Zhuyin) Display(input string) string {
	var b strings.Builder
	for _, k := range input {
		if s, ok := dachen[k]; ok {
			b.WriteRune(s)
		} else if t, ok := tones[k]; ok {
			b.WriteRune(t)
		}
	}
	return b.String()
}
