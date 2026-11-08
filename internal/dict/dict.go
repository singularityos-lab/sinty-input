package dict

import (
	"bufio"
	"compress/gzip"
	"embed"
	"strconv"
	"strings"
	"sync"
)

//go:embed data/*.txt.gz
var data embed.FS

type Table struct {
	once    sync.Once
	file    string
	entries map[string][]string
	scores  map[string][]int
	longest int
}

var (
	Japanese = &Table{file: "data/ja.txt.gz"}
	Chinese  = &Table{file: "data/zh.txt.gz"}
)

func (t *Table) load() {
	t.entries, t.scores = map[string][]string{}, map[string][]int{}
	f, err := data.Open(t.file)
	if err != nil {
		return
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return
	}
	s := bufio.NewScanner(z)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	for s.Scan() {
		fields := strings.Split(s.Text(), "\t")
		if len(fields) < 2 {
			continue
		}
		key := fields[0]
		t.entries[key] = strings.Split(fields[1], "/")
		if len(fields) > 2 {
			for _, n := range strings.Split(fields[2], ",") {
				v, _ := strconv.Atoi(n)
				t.scores[key] = append(t.scores[key], v)
			}
		}
		if n := len([]rune(key)); n > t.longest {
			t.longest = n
		}
	}
}

func (t *Table) Lookup(key string) []string {
	t.once.Do(t.load)
	return t.entries[key]
}

func (t *Table) Score(key string) int {
	t.once.Do(t.load)
	if s := t.scores[key]; len(s) > 0 {
		return s[0]
	}
	return 0
}

func (t *Table) Longest() int {
	t.once.Do(t.load)
	return t.longest
}

func (t *Table) Each(fn func(key string, values []string)) {
	t.once.Do(t.load)
	for k, v := range t.entries {
		fn(k, v)
	}
}
