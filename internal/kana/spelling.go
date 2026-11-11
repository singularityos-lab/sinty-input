package kana

import (
	"strings"

	"github.com/singularityos-lab/sinty-input/internal/dict"
)

type Spelling struct{}

func (Spelling) Key(r rune, input string) bool {
	if r >= 'a' && r <= 'z' || r == '-' {
		return true
	}
	return input != "" && (r == '\'' || prefixes[string(r)])
}

func (Spelling) Segment(input string) ([]string, []int, bool) {
	var segs []string
	var ends []int
	i := 0
	for i < len(input) {
		if input[i] == '\'' && (i == 0 || input[i-1] != 'n') {
			i++
			continue
		}
		n := 0
		for l := min(4, len(input)-i); l >= 1; l-- {
			if _, ok := romaji[input[i:i+l]]; ok {
				n = l
				break
			}
		}
		rest := input[i:]
		switch {
		case n > 0:
			segs = append(segs, romaji[input[i:i+n]])
		case rest == "n":
			segs, n = append(segs, "ん"), 1
		case prefixes[rest]:
			segs = append(segs, rest)
			ends = append(ends, len(input))
			return segs, ends, false
		case len(rest) >= 2 && rest[0] == 'n' && !vowel(rest[1]) && rest[1] != 'y':
			segs, n = append(segs, "ん"), 1
		case len(rest) >= 2 && (rest[0] == rest[1] || rest[0] == 't' && rest[1] == 'c') && !vowel(rest[0]):
			segs, n = append(segs, "っ"), 1
		default:
			segs, n = append(segs, rest[:1]), 1
		}
		i += n
		ends = append(ends, i)
	}
	return segs, ends, true
}

func (Spelling) Expand(string) []string {
	return nil
}

func (s Spelling) Display(input string) string {
	segs, _, _ := s.Segment(input)
	return strings.Join(segs, "")
}

func (s Spelling) Literal(input string) string {
	segs, _, complete := s.Segment(input)
	if !complete {
		segs[len(segs)-1] = finish(segs[len(segs)-1])
	}
	return strings.Join(segs, "")
}

func Rank(d *dict.Dict, reading string, whole []string) ([]string, bool) {
	var inflected []string
	runes := []rune(reading)
	for n := 1; n <= 4 && n < len(runes); n++ {
		stem, tail := string(runes[:len(runes)-n]), string(runes[len(runes)-n:])
		c := okuri(runes[len(runes)-n])
		if c == "" {
			continue
		}
		for _, w := range d.Words(stem + c) {
			inflected = append(inflected, w+tail)
		}
	}
	lead, more := min(3, len(whole)), min(6, len(inflected))
	var out []string
	for _, group := range [][]string{whole[:lead], inflected[:more], whole[lead:], inflected[more:]} {
		out = append(out, group...)
	}
	return append(out, reading, katakana(reading)), len(inflected) > 0
}
