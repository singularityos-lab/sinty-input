package phonetic

import (
	"sort"
	"strings"
)

type Layout struct {
	Name     string
	Initials map[string]byte
	Finals   map[string]byte
	Prefix   bool
	Double   bool
}

func finals(pairs string) map[string]byte {
	m := map[string]byte{}
	for _, f := range strings.Fields(pairs) {
		k, v, _ := strings.Cut(f, "=")
		m[k] = v[0]
	}
	return m
}

var Layouts = []Layout{
	{"xiaohe", map[string]byte{"zh": 'v', "ch": 'i', "sh": 'u'},
		finals("iu=q ei=w uan=r van=r ue=t ve=t un=y vn=y uo=o ie=p ong=s iong=s ai=d en=f eng=g ang=h an=j ing=k uai=k iang=l uang=l ou=z ia=x ua=x ao=c ui=v v=v in=b iao=n ian=m"),
		false, true},
	{"ziranma", map[string]byte{"zh": 'v', "ch": 'i', "sh": 'u'},
		finals("iu=q ia=w ua=w uan=r van=r ue=t ve=t ing=y uai=y uo=o un=p vn=p ong=s iong=s iang=d uang=d en=f eng=g ang=h an=j ao=k ai=l ei=z ie=x iao=c ui=v v=v ou=b in=n ian=m"),
		false, true},
	{"microsoft", map[string]byte{"zh": 'v', "ch": 'i', "sh": 'u'},
		finals("ai=l an=j ang=h ao=k ei=z en=f eng=g er=r ia=w ian=m iang=d iao=c ie=x in=n ing=; iong=s iu=q ong=s ou=b ua=w uai=y uan=r van=r uang=d ue=t ui=v un=p vn=p uo=o ve=v v=y"),
		true, true},
	{"sogou", map[string]byte{"zh": 'v', "ch": 'i', "sh": 'u'},
		finals("ai=l an=j ang=h ao=k ei=z en=f eng=g er=r ia=w ian=m iang=d iao=c ie=x in=n ing=; iong=s iu=q ong=s ou=b ua=w uai=y uan=r van=r uang=d ue=t ui=v un=p vn=p uo=o ve=t v=y"),
		true, false},
	{"abc", map[string]byte{"zh": 'a', "ch": 'e', "sh": 'v'},
		finals("ai=l an=j ang=h ao=k ei=q en=f eng=g er=r ia=d ian=w iang=t iao=z ie=x in=c ing=y iong=s iu=r ong=s ou=b ua=d uai=c uan=p van=p uang=t ue=m ui=m un=n vn=n uo=o ve=m v=v"),
		true, false},
	{"ziguang", map[string]byte{"zh": 'u', "ch": 'a', "sh": 'i'},
		finals("ai=p an=r ang=s ao=q ei=k en=w eng=t er=j ia=x ian=f iang=g iao=b ie=d in=y ing=; iong=h iu=j ong=h ou=z ua=x uai=y uan=l van=l uang=g ue=n ui=n un=m vn=m uo=o ve=n v=v"),
		true, false},
}

func LayoutByName(name string) (Layout, bool) {
	for _, l := range Layouts {
		if l.Name == name {
			return l, true
		}
	}
	return Layout{}, false
}

var pinyinInitials = []string{"zh", "ch", "sh", "b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h", "j", "q", "x", "r", "z", "c", "s", "y", "w"}

func SplitPinyin(s string) (string, string) {
	for _, i := range pinyinInitials {
		if strings.HasPrefix(s, i) && len(s) > len(i) {
			return i, s[len(i):]
		}
	}
	return "", s
}

func (l Layout) final(f string) (byte, bool) {
	if k, ok := l.Finals[f]; ok {
		return k, true
	}
	if len(f) == 1 && strings.Contains("aoeiu", f) {
		return f[0], true
	}
	return 0, false
}

func (l Layout) Codes(syllable string) []string {
	initial, rest := SplitPinyin(syllable)
	if initial == "" {
		var out []string
		first := rest[0]
		if !strings.ContainsRune("aoe", rune(first)) {
			return nil
		}
		k, ok := l.final(rest)
		if l.Prefix && ok {
			out = append(out, "o"+string(k))
		}
		if l.Double || !l.Prefix {
			switch {
			case len(rest) == 1:
				out = append(out, rest+rest)
			case len(rest) == 2:
				out = append(out, rest)
				if ok {
					out = append(out, string(first)+string(k))
				}
			case ok:
				out = append(out, string(first)+string(k))
			}
		}
		return out
	}
	ik := initial[0]
	if k, ok := l.Initials[initial]; ok {
		ik = k
	} else if len(initial) > 1 {
		return nil
	}
	var out []string
	add := func(f string) {
		if k, ok := l.final(f); ok {
			out = append(out, string(ik)+string(k))
		}
	}
	add(rest)
	if strings.ContainsRune("jqxy", rune(initial[0])) && strings.HasPrefix(rest, "u") {
		add("v" + rest[1:])
	}
	return out
}

type Shuangpin struct {
	layout    Layout
	syllables map[string]string
	byKey     map[byte][]string
}

func NewShuangpin(l Layout, syllables map[string]bool) *Shuangpin {
	s := &Shuangpin{layout: l, syllables: map[string]string{}, byKey: map[byte][]string{}}
	names := make([]string, 0, len(syllables))
	for syl := range syllables {
		names = append(names, syl)
	}
	sort.Strings(names)
	for _, syl := range names {
		for _, c := range l.Codes(syl) {
			if old, taken := s.syllables[c]; !taken || len(syl) > len(old) {
				s.syllables[c] = syl
			}
		}
	}
	for c, syl := range s.syllables {
		s.byKey[c[0]] = append(s.byKey[c[0]], syl)
	}
	for _, list := range s.byKey {
		sort.Strings(list)
	}
	return s
}

func (s *Shuangpin) Table() map[string]string {
	return s.syllables
}

func (s *Shuangpin) Key(r rune, input string) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	return r == ';' && len(input)%2 == 1
}

func (s *Shuangpin) Segment(input string) ([]string, []int, bool) {
	var segs []string
	var ends []int
	for i := 0; i < len(input); i += 2 {
		if i+1 == len(input) {
			segs = append(segs, input[i:])
			ends = append(ends, len(input))
			return segs, ends, false
		}
		syl, ok := s.syllables[input[i:i+2]]
		if !ok {
			segs = append(segs, input[i:])
			ends = append(ends, len(input))
			return segs, ends, false
		}
		segs = append(segs, syl)
		ends = append(ends, i+2)
	}
	return segs, ends, true
}

func (s *Shuangpin) Expand(partial string) []string {
	if len(partial) != 1 {
		return nil
	}
	return s.byKey[partial[0]]
}

func (s *Shuangpin) Display(input string) string {
	segs, _, _ := s.Segment(input)
	return strings.Join(segs, "'")
}
