package kana

import "strings"

var romaji = map[string]string{
	"a": "あ", "i": "い", "u": "う", "e": "え", "o": "お",
	"ka": "か", "ki": "き", "ku": "く", "ke": "け", "ko": "こ",
	"ga": "が", "gi": "ぎ", "gu": "ぐ", "ge": "げ", "go": "ご",
	"sa": "さ", "si": "し", "shi": "し", "su": "す", "se": "せ", "so": "そ",
	"za": "ざ", "zi": "じ", "ji": "じ", "zu": "ず", "ze": "ぜ", "zo": "ぞ",
	"ta": "た", "ti": "ち", "chi": "ち", "tu": "つ", "tsu": "つ", "te": "て", "to": "と",
	"da": "だ", "di": "ぢ", "du": "づ", "de": "で", "do": "ど",
	"na": "な", "ni": "に", "nu": "ぬ", "ne": "ね", "no": "の",
	"ha": "は", "hi": "ひ", "hu": "ふ", "fu": "ふ", "he": "へ", "ho": "ほ",
	"ba": "ば", "bi": "び", "bu": "ぶ", "be": "べ", "bo": "ぼ",
	"pa": "ぱ", "pi": "ぴ", "pu": "ぷ", "pe": "ぺ", "po": "ぽ",
	"ma": "ま", "mi": "み", "mu": "む", "me": "め", "mo": "も",
	"ya": "や", "yu": "ゆ", "yo": "よ",
	"ra": "ら", "ri": "り", "ru": "る", "re": "れ", "ro": "ろ",
	"la": "ら", "li": "り", "lu": "る", "le": "れ", "lo": "ろ",
	"wa": "わ", "wi": "うぃ", "we": "うぇ", "wo": "を",
	"nn": "ん", "n'": "ん", "xn": "ん",
	"va": "ゔぁ", "vi": "ゔぃ", "vu": "ゔ", "ve": "ゔぇ", "vo": "ゔぉ",
	"fa": "ふぁ", "fi": "ふぃ", "fe": "ふぇ", "fo": "ふぉ",
	"ja": "じゃ", "ju": "じゅ", "je": "じぇ", "jo": "じょ",
	"sha": "しゃ", "shu": "しゅ", "she": "しぇ", "sho": "しょ",
	"cha": "ちゃ", "chu": "ちゅ", "che": "ちぇ", "cho": "ちょ",
	"tsa": "つぁ", "thi": "てぃ", "dhi": "でぃ", "twu": "とぅ", "dwu": "どぅ",
	"xa": "ぁ", "xi": "ぃ", "xu": "ぅ", "xe": "ぇ", "xo": "ぉ",
	"xya": "ゃ", "xyu": "ゅ", "xyo": "ょ", "xtu": "っ", "xtsu": "っ", "xwa": "ゎ",
	"ltu": "っ", "lya": "ゃ", "lyu": "ゅ", "lyo": "ょ",
	"-": "ー", ",": "、", ".": "。", "[": "「", "]": "」",
	"?": "？", "!": "！", "~": "〜", "/": "・",
}

func init() {
	for c, row := range map[string]string{
		"k": "き", "g": "ぎ", "s": "し", "z": "じ", "t": "ち", "d": "ぢ", "n": "に",
		"h": "ひ", "b": "び", "p": "ぴ", "m": "み", "r": "り", "c": "ち", "j": "じ",
	} {
		for v, small := range map[string]string{"a": "ゃ", "u": "ゅ", "o": "ょ", "e": "ぇ"} {
			romaji[c+"y"+v] = row + small
		}
		romaji[c+"yi"] = row + "ぃ"
	}
	for k := range romaji {
		for i := 1; i <= len(k); i++ {
			prefixes[k[:i]] = true
		}
	}
}

var prefixes = map[string]bool{}

func vowel(c byte) bool {
	return strings.IndexByte("aiueo", c) >= 0
}

func convert(pending string) (string, string) {
	var out strings.Builder
	for pending != "" {
		if kana, ok := romaji[pending]; ok {
			out.WriteString(kana)
			return out.String(), ""
		}
		if prefixes[pending] {
			break
		}
		c := pending[0]
		switch {
		case len(pending) >= 2 && c == 'n' && !vowel(pending[1]) && pending[1] != 'y':
			out.WriteString("ん")
		case len(pending) >= 2 && (c == pending[1] || c == 't' && pending[1] == 'c') && !vowel(c):
			out.WriteString("っ")
		default:
			out.WriteByte(c)
		}
		pending = pending[1:]
	}
	return out.String(), pending
}

func finish(pending string) string {
	if pending == "n" {
		return "ん"
	}
	return pending
}

var rows = map[string]string{
	"k": "かきくけこ", "g": "がぎぐげご", "s": "さしすせそ", "z": "ざじずぜぞ",
	"t": "たちつてとっ", "d": "だぢづでど", "n": "なにぬねのん", "h": "はひふへほ",
	"b": "ばびぶべぼ", "p": "ぱぴぷぺぽ", "m": "まみむめも", "y": "やゆよ",
	"r": "らりるれろ", "w": "わを", "a": "あ", "i": "い", "u": "う", "e": "え", "o": "お",
}

func okuri(kana rune) string {
	for c, row := range rows {
		if strings.ContainsRune(row, kana) {
			return c
		}
	}
	return ""
}

func katakana(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'ぁ' && r <= 'ゖ' {
			return r + 0x60
		}
		return r
	}, s)
}
