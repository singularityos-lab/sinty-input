package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/dict/importer"
)

func save(d *dict.Dict, out, name string) error {
	log.Printf("%s: %d entries", name, d.Len())
	return d.Save(filepath.Join(out, name+dict.ExtGz))
}

func kanaKey(key string, okuri bool) bool {
	runes := []rune(key)
	if okuri {
		last := runes[len(runes)-1]
		if last < 'a' || last > 'z' {
			return false
		}
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes {
		if !(r >= 'ぁ' && r <= 'ゖ') && r != 'ー' {
			return false
		}
	}
	return true
}

func cleanCandidate(c string) string {
	if i := strings.IndexRune(c, ';'); i >= 0 {
		c = c[:i]
	}
	if c == "" || strings.HasPrefix(c, "(") || strings.ContainsAny(c, "/\t\n") {
		return ""
	}
	return c
}

func japanese(src, out string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	var skk struct {
		OkuriAri  map[string][]string `json:"okuri_ari"`
		OkuriNasi map[string][]string `json:"okuri_nasi"`
	}
	if err := json.Unmarshal(data, &skk); err != nil {
		return err
	}
	values := map[string][]string{}
	add := func(entries map[string][]string, okuri bool) {
		for k, cs := range entries {
			if !kanaKey(k, okuri) {
				continue
			}
			for _, c := range cs {
				if c = cleanCandidate(c); c != "" {
					values[k] = append(values[k], c)
				}
			}
		}
	}
	add(skk.OkuriNasi, false)
	add(skk.OkuriAri, true)
	keys := make([]string, 0, len(values))
	for k, v := range values {
		if len(v) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	d := dict.New("", map[string]string{"name": "Japanese (Kana)", "type": "phonetic", "language": "ja",
		"license": "GPL-2.0-or-later", "source": "https://github.com/skk-dev/dict"})
	for _, k := range keys {
		for _, v := range values[k] {
			d.Add(k, v, 0)
		}
	}
	return save(d, out, "ja-kana")
}

var cedictLine = regexp.MustCompile(`^(\S+) (\S+) \[([^\]]+)\] /(.*)/$`)

func frequencies(path string) (map[rune]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	freq := map[rune]float64{}
	number := regexp.MustCompile(`\((\d+)\)`)
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Split(s.Text(), "\t")
		if len(fields) != 3 || fields[1] != "kHanyuPinlu" {
			continue
		}
		cp, err := strconv.ParseUint(strings.TrimPrefix(fields[0], "U+"), 16, 32)
		if err != nil {
			continue
		}
		total := 0.0
		for _, m := range number.FindAllStringSubmatch(fields[2], -1) {
			n, _ := strconv.ParseFloat(m[1], 64)
			total += n
		}
		freq[rune(cp)] = total
	}
	return freq, s.Err()
}

func syllables(pinyin string) ([]string, bool) {
	var out []string
	for _, s := range strings.Fields(strings.ToLower(pinyin)) {
		s = strings.ReplaceAll(s, "u:", "v")
		s = strings.TrimRight(s, "012345")
		if s == "" {
			return nil, false
		}
		for _, r := range s {
			if r < 'a' || r > 'z' {
				return nil, false
			}
		}
		out = append(out, s)
	}
	return out, len(out) > 0
}

func han(word string) bool {
	for _, r := range word {
		if !unicode.Is(unicode.Han, r) && r != '〇' {
			return false
		}
	}
	return true
}

func wordFrequencies(path string) (map[string]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	freq := map[string]float64{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 2 {
			continue
		}
		n, err := strconv.ParseFloat(fields[1], 64)
		if err == nil {
			freq[fields[0]] += n
		}
	}
	return freq, s.Err()
}

func chinese(cedict, unihan, words, out string) error {
	freq, err := frequencies(unihan)
	if err != nil {
		return err
	}
	wordFreq, err := wordFrequencies(words)
	if err != nil {
		return err
	}
	f, err := os.Open(cedict)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(cedict, ".gz") {
		z, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		r = z
	}
	type ranked struct {
		word  string
		score float64
		order int
	}
	entries := map[string][]ranked{}
	seen := map[string]bool{}
	order := 0
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1<<20), 1<<20)
	for s.Scan() {
		m := cedictLine.FindStringSubmatch(s.Text())
		if m == nil {
			continue
		}
		simp, pinyin, defs := m[2], m[3], m[4]
		if !han(simp) || strings.HasPrefix(defs, "variant of") || strings.HasPrefix(defs, "old variant of") {
			continue
		}
		syl, ok := syllables(pinyin)
		if !ok || len(syl) != len([]rune(simp)) {
			continue
		}
		key := strings.Join(syl, " ")
		if seen[key+"\t"+simp] {
			continue
		}
		seen[key+"\t"+simp] = true
		score := math.Log1p(wordFreq[simp])
		if score == 0 && len([]rune(simp)) == 1 {
			score = math.Log1p(freq[[]rune(simp)[0]]) / 2
		}
		order++
		entries[key] = append(entries[key], ranked{simp, score, order})
	}
	if err := s.Err(); err != nil {
		return err
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	d := dict.New("", map[string]string{"name": "Chinese (Pinyin)", "type": "phonetic", "language": "zh-Hans",
		"license": "CC-BY-SA-4.0", "source": "https://www.mdbg.net/chinese/dictionary?page=cc-cedict"})
	for _, k := range keys {
		list := entries[k]
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].score != list[j].score {
				return list[i].score > list[j].score
			}
			return list[i].order < list[j].order
		})
		for _, w := range list {
			d.Add(k, w.word, max(0, int(math.Round(w.score*10))))
		}
	}
	return save(d, out, "zh-pinyin")
}

type table struct {
	file, format, name, title, language, keys, length, wildcard, autocommit, license, source string
}

var tables = []table{
	{"wubi86.dict.yaml", "rime", "zh-wubi86", "Wubi 86", "zh-Hans", "abcdefghijklmnopqrstuvwxy", "4", "z", "yes", "LGPL-3.0-only", "https://github.com/rime/rime-wubi"},
	{"wubi98.txt", "fcitx", "zh-wubi98", "Wubi 98", "zh-Hans", "", "", "z", "yes", "LicenseRef-public-domain", "https://github.com/fcitx/fcitx5-table-extra"},
	{"xinshijiwubi.dict.yaml", "rime", "zh-wubi-ng", "Wubi New Generation", "zh-Hans", "abcdefghijklmnopqrstuvwxy", "4", "z", "yes", "LGPL-3.0-only", "https://github.com/Pioneer-Linzi/xinshijiwubi"},
	{"cangjie3.txt", "fcitx", "zh-cangjie3", "Cangjie 3", "zh-Hant", "", "", "*", "no", "LicenseRef-public-domain", "https://github.com/fcitx/fcitx5-table-extra"},
	{"cangjie5.txt", "fcitx", "zh-cangjie5", "Cangjie 5", "zh-Hant", "", "", "*", "no", "LicenseRef-public-domain", "https://github.com/fcitx/fcitx5-table-extra"},
	{"quick5.txt", "fcitx", "zh-quick", "Quick", "zh-Hant", "", "", "*", "no", "LicenseRef-public-domain", "https://github.com/fcitx/fcitx5-table-extra"},
	{"array30.txt", "fcitx", "zh-array30", "Array 30", "zh-Hant", "", "", "*", "no", "LicenseRef-public-domain", "https://github.com/fcitx/fcitx5-table-extra"},
	{"zhengma.txt", "fcitx", "zh-zhengma", "Zhengma", "zh-Hans", "", "", "*", "yes", "LicenseRef-public-domain", "https://github.com/fcitx/fcitx5-table-extra"},
}

func importTable(src string, t table, out string) error {
	f, err := os.Open(filepath.Join(src, t.file))
	if err != nil {
		return err
	}
	defer f.Close()
	var d *dict.Dict
	if t.format == "rime" {
		d, err = importer.Rime(f)
	} else {
		d, err = importer.Fcitx(f)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", t.file, err)
	}
	delete(d.Meta, "rime")
	set := func(k, v string) {
		if v != "" {
			d.Meta[k] = v
		}
	}
	set("name", t.title)
	set("type", "table")
	set("language", t.language)
	set("keys", t.keys)
	set("length", t.length)
	set("wildcard", t.wildcard)
	set("autocommit", t.autocommit)
	set("license", t.license)
	set("source", t.source)
	if t.format == "rime" {
		var drop []dict.Entry
		d.Each(func(code string, list []dict.Entry) {
			if strings.HasPrefix(code, "z") {
				drop = append(drop, list...)
			}
		})
		for _, e := range drop {
			d.Remove(e.Code, e.Text)
		}
	}
	return save(d, out, t.name)
}

func toneless(code string) string {
	parts := strings.Fields(code)
	for i, p := range parts {
		parts[i] = strings.TrimRight(p, "0123456789")
	}
	return strings.Join(parts, " ")
}

func cantonese(src, out string) error {
	freq, err := wordFrequencies(filepath.Join(src, "essay-cantonese.txt"))
	if err != nil {
		return err
	}
	d := dict.New("", map[string]string{"name": "Cantonese (Jyutping)", "type": "phonetic", "language": "yue",
		"license": "CC-BY-4.0", "source": "https://github.com/rime/rime-cantonese"})
	for _, file := range []string{"jyut6ping3.chars.dict.yaml", "jyut6ping3.words.dict.yaml"} {
		f, err := os.Open(filepath.Join(src, file))
		if err != nil {
			return err
		}
		part, err := importer.Rime(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		part.Each(func(code string, list []dict.Entry) {
			key := toneless(code)
			for _, e := range list {
				if strings.ContainsAny(key, "0123456789") || key == "" {
					continue
				}
				w := max(0, int(math.Round(math.Log1p(freq[e.Text])*10)))
				if e.Weight > 0 && w == 0 {
					w = 1
				}
				d.Add(key, e.Text, w)
			}
		})
	}
	return save(d, out, "yue-jyutping")
}

type conversion struct {
	name, title string
	files       []string
}

var conversions = []conversion{
	{"zh-s2t", "Simplified to Traditional", []string{"STPhrases.txt", "STCharacters.txt"}},
	{"zh-t2s", "Traditional to Simplified", []string{"TSPhrases.txt", "TSCharacters.txt"}},
	{"zh-tw-variants", "Taiwan Variants", []string{"TWVariants.txt"}},
	{"zh-tw-phrases", "Taiwan Phrases", []string{"TWPhrases.txt"}},
	{"zh-hk-variants", "Hong Kong Variants", []string{"HKVariants.txt"}},
}

func openccTable(src string, c conversion, out string) error {
	d := dict.New("", map[string]string{"name": c.title, "type": "conversion", "language": "zh",
		"license": "Apache-2.0", "source": "https://github.com/BYVoid/OpenCC"})
	for _, file := range c.files {
		f, err := os.Open(filepath.Join(src, file))
		if err != nil {
			return err
		}
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 1<<16), 1<<20)
		for s.Scan() {
			line := s.Text()
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "\t")
			if !ok {
				continue
			}
			values := strings.Fields(v)
			for i, x := range values {
				d.Add(k, x, len(values)-i)
			}
		}
		f.Close()
		if err := s.Err(); err != nil {
			return err
		}
	}
	return save(d, out, c.name)
}

func main() {
	src := flag.String("sources", "sources", "directory with the downloaded sources")
	out := flag.String("out", "dict", "output directory")
	only := flag.String("only", "", "comma separated dictionary names to build")
	flag.Parse()
	want := func(name string) bool {
		if *only == "" {
			return true
		}
		for _, n := range strings.Split(*only, ",") {
			if n == name {
				return true
			}
		}
		return false
	}
	run := func(name string, fn func() error) {
		if !want(name) {
			return
		}
		if err := fn(); err != nil {
			log.Fatalf("%s: %v", name, err)
		}
	}
	run("ja-kana", func() error { return japanese(filepath.Join(*src, "SKK-JISYO.L.json"), *out) })
	run("zh-pinyin", func() error {
		return chinese(filepath.Join(*src, "cedict.txt.gz"), filepath.Join(*src, "Unihan_Readings.txt"), filepath.Join(*src, "dict.txt"), *out)
	})
	for _, t := range tables {
		run(t.name, func() error { return importTable(*src, t, *out) })
	}
	run("yue-jyutping", func() error { return cantonese(*src, *out) })
	for _, c := range conversions {
		run(c.name, func() error { return openccTable(*src, c, *out) })
	}
}
