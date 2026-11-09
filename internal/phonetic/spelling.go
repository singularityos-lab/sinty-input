package phonetic

import (
	"sort"
	"strings"
)

type Spelling interface {
	Key(r rune, input string) bool
	Segment(input string) (segs []string, ends []int, complete bool)
	Expand(partial string) []string
	Display(input string) string
}

type Alphabetic struct {
	syllables map[string]bool
	partials  map[string][]string
	longest   int
}

func NewAlphabetic(syllables map[string]bool) *Alphabetic {
	a := &Alphabetic{syllables: syllables, partials: map[string][]string{}}
	for s := range syllables {
		a.longest = max(a.longest, len(s))
		for i := 1; i < len(s); i++ {
			a.partials[s[:i]] = append(a.partials[s[:i]], s)
		}
	}
	for _, list := range a.partials {
		sort.Strings(list)
	}
	return a
}

func (a *Alphabetic) Key(r rune, input string) bool {
	return (r >= 'a' && r <= 'z') || (r == '\'' && input != "")
}

func (a *Alphabetic) Segment(input string) ([]string, []int, bool) {
	var segs []string
	var ends []int
	offset := 0
	for _, part := range strings.Split(input, "'") {
		if part != "" {
			parts, ok := a.split(part)
			pos := offset
			for _, p := range parts {
				pos += len(p)
				segs = append(segs, p)
				ends = append(ends, pos)
			}
			if !ok {
				return segs, ends, false
			}
		}
		offset += len(part) + 1
	}
	return segs, ends, true
}

func (a *Alphabetic) split(s string) ([]string, bool) {
	n := len(s)
	best := make([]int, n+1)
	from := make([]int, n+1)
	for i := 1; i <= n; i++ {
		best[i] = -1
		for l := min(a.longest, i); l >= 1; l-- {
			if best[i-l] >= 0 && a.syllables[s[i-l:i]] && (best[i] < 0 || best[i-l]+1 < best[i]) {
				best[i], from[i] = best[i-l]+1, i-l
			}
		}
	}
	if best[n] >= 0 {
		return walk(s, from, n), true
	}
	for i := n - 1; i >= 0; i-- {
		if best[i] >= 0 && a.partials[s[i:]] != nil {
			return append(walk(s, from, i), s[i:]), false
		}
	}
	return []string{s}, false
}

func walk(s string, from []int, end int) []string {
	var out []string
	for end > 0 {
		out = append([]string{s[from[end]:end]}, out...)
		end = from[end]
	}
	return out
}

func (a *Alphabetic) Expand(partial string) []string {
	return a.partials[partial]
}

func (a *Alphabetic) Display(input string) string {
	segs, _, _ := a.Segment(input)
	return strings.Join(segs, "'")
}
