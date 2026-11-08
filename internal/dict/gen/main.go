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
)

func writeTable(path string, keys []string, values map[string][]string, scores map[string][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	z, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(z)
	for _, k := range keys {
		if scores != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\n", k, strings.Join(values[k], "/"), strings.Join(scores[k], ","))
		} else {
			fmt.Fprintf(w, "%s\t%s\n", k, strings.Join(values[k], "/"))
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return z.Close()
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
	log.Printf("ja: %d readings", len(keys))
	return writeTable(out, keys, values, nil)
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
	values, scores := map[string][]string{}, map[string][]string{}
	keys := make([]string, 0, len(entries))
	for k, list := range entries {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].score != list[j].score {
				return list[i].score > list[j].score
			}
			return list[i].order < list[j].order
		})
		for _, w := range list {
			values[k] = append(values[k], w.word)
			scores[k] = append(scores[k], strconv.Itoa(max(0, int(math.Round(w.score*10)))))
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	log.Printf("zh: %d readings", len(keys))
	return writeTable(out, keys, values, scores)
}

func main() {
	skk := flag.String("skk", "", "SKK-JISYO.L.json")
	cedict := flag.String("cedict", "", "cedict_1_0_ts_utf-8_mdbg.txt.gz")
	unihan := flag.String("unihan", "", "Unihan_Readings.txt")
	words := flag.String("words", "", "jieba dict.txt")
	out := flag.String("out", "internal/dict/data", "output directory")
	flag.Parse()
	if *skk != "" {
		if err := japanese(*skk, filepath.Join(*out, "ja.txt.gz")); err != nil {
			log.Fatal(err)
		}
	}
	if *cedict != "" {
		if err := chinese(*cedict, *unihan, *words, filepath.Join(*out, "zh.txt.gz")); err != nil {
			log.Fatal(err)
		}
	}
}
