package punct

import "strings"

var dash = strings.Repeat(string(rune(0x2014)), 2)

var simplified = map[rune]string{
	',': "，", '.': "。", '?': "？", '!': "！", ':': "：", ';': "；",
	'(': "（", ')': "）", '[': "【", ']': "】", '<': "《", '>': "》",
	'\\': "、", '~': "～", '$': "￥", '^': "……", '_': dash,
}

var traditional = map[rune]string{
	',': "，", '.': "。", '?': "？", '!': "！", ':': "：", ';': "；",
	'(': "（", ')': "）", '[': "「", ']': "」", '{': "『", '}': "』", '<': "《", '>': "》",
	'\\': "、", '~': "～", '$': "＄", '^': "……", '_': dash,
}

var japanese = map[rune]string{
	',': "、", '.': "。", '?': "？", '!': "！", ':': "：", ';': "；",
	'(': "（", ')': "）", '[': "「", ']': "」", '{': "『", '}': "』",
	'~': "〜", '/': "・", '-': "ー",
}

var korean = map[rune]string{
	'~': "～",
}

func Table(language string) map[rune]string {
	switch {
	case language == "zh-Hant" || language == "yue" || strings.HasPrefix(language, "zh-TW") || strings.HasPrefix(language, "zh-HK"):
		return traditional
	case strings.HasPrefix(language, "zh"):
		return simplified
	case language == "ja":
		return japanese
	case language == "ko":
		return korean
	}
	return nil
}

func Width(r rune) string {
	switch {
	case r == ' ':
		return "　"
	case r > ' ' && r <= '~':
		return string(r - '!' + '！')
	}
	return string(r)
}

type Style struct {
	Language  string
	FullWidth bool
}

func (s Style) Map(r rune) (string, bool) {
	if s.FullWidth {
		if p, ok := Table(s.Language)[r]; ok {
			return p, true
		}
		if r >= '!' && r <= '~' && !(r >= '0' && r <= '9') && !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') {
			if Table(s.Language) != nil {
				return Width(r), true
			}
		}
	}
	return "", false
}
