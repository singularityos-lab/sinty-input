package dict

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const Format = "sinty-dict 1"

type Entry struct {
	Code   string
	Text   string
	Weight int
}

type Dict struct {
	Name    string
	Path    string
	Meta    map[string]string
	order   []string
	entries map[string][]Entry
	codes   []string
	longest int
	dirty   bool
}

func New(name string, meta map[string]string) *Dict {
	if meta == nil {
		meta = map[string]string{}
	}
	return &Dict{Name: name, Meta: meta, entries: map[string][]Entry{}}
}

func open(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(path, ".gz") {
		return f, nil
	}
	z, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return struct {
		io.Reader
		io.Closer
	}{z, f}, nil
}

func Load(path string) (*Dict, error) {
	r, err := open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	d, err := Read(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	d.Name, d.Path = NameOf(path), path
	return d, nil
}

func ReadHeader(path string) (map[string]string, error) {
	r, err := open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	meta := map[string]string{}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "@") {
			break
		}
		k, v := header(line)
		meta[k] = v
	}
	if meta["format"] != Format {
		return nil, errors.New("not a sinty-input dictionary")
	}
	return meta, s.Err()
}

func header(line string) (string, string) {
	k, v, _ := strings.Cut(strings.TrimPrefix(line, "@"), " ")
	return strings.TrimSpace(k), strings.TrimSpace(v)
}

func Read(r io.Reader) (*Dict, error) {
	d := New("", nil)
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	body := false
	n := 0
	for s.Scan() {
		n++
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !body && strings.HasPrefix(line, "@") {
			k, v := header(line)
			d.Meta[k] = v
			continue
		}
		if !body && d.Meta["format"] != Format {
			return nil, errors.New("not a sinty-input dictionary")
		}
		body = true
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || fields[0] == "" || fields[1] == "" {
			return nil, fmt.Errorf("line %d: expected code, text and an optional weight", n)
		}
		w := 0
		if len(fields) > 2 && fields[2] != "" {
			v, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: weight %q", n, fields[2])
			}
			w = v
		}
		d.add(Entry{fields[0], fields[1], w})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if d.Meta["format"] != Format {
		return nil, errors.New("not a sinty-input dictionary")
	}
	d.finish()
	return d, nil
}

func (d *Dict) add(e Entry) {
	list, ok := d.entries[e.Code]
	if !ok {
		d.order = append(d.order, e.Code)
	}
	for i, x := range list {
		if x.Text == e.Text {
			if e.Weight > x.Weight {
				list[i].Weight = e.Weight
			}
			return
		}
	}
	d.entries[e.Code] = append(list, e)
}

func (d *Dict) sync() {
	if d.dirty {
		d.finish()
	}
}

func (d *Dict) finish() {
	d.dirty = false
	d.codes = make([]string, 0, len(d.entries))
	d.longest = 0
	for code, list := range d.entries {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Weight > list[j].Weight })
		d.codes = append(d.codes, code)
		d.longest = max(d.longest, len([]rune(code)))
	}
	sort.Strings(d.codes)
}

func (d *Dict) Add(code, text string, weight int) {
	d.add(Entry{code, text, weight})
	d.dirty = true
}

func (d *Dict) Set(code, text string, weight int) {
	list := d.entries[code]
	for i, x := range list {
		if x.Text == text {
			list[i].Weight = weight
			d.dirty = true
			return
		}
	}
	d.Add(code, text, weight)
}

func (d *Dict) Remove(code, text string) bool {
	list := d.entries[code]
	for i, x := range list {
		if x.Text != text {
			continue
		}
		list = append(list[:i:i], list[i+1:]...)
		if len(list) == 0 {
			delete(d.entries, code)
			for j, c := range d.order {
				if c == code {
					d.order = append(d.order[:j:j], d.order[j+1:]...)
					break
				}
			}
		} else {
			d.entries[code] = list
		}
		d.dirty = true
		return true
	}
	return false
}

func (d *Dict) Lookup(code string) []Entry {
	if d == nil {
		return nil
	}
	d.sync()
	return d.entries[code]
}

func (d *Dict) Words(code string) []string {
	list := d.Lookup(code)
	if list == nil {
		return nil
	}
	out := make([]string, len(list))
	for i, e := range list {
		out[i] = e.Text
	}
	return out
}

func (d *Dict) Score(code string) int {
	if list := d.Lookup(code); len(list) > 0 {
		return list[0].Weight
	}
	return 0
}

func (d *Dict) Weight(code, text string) (int, bool) {
	for _, e := range d.Lookup(code) {
		if e.Text == text {
			return e.Weight, true
		}
	}
	return 0, false
}

func (d *Dict) Longest() int {
	if d == nil {
		return 0
	}
	d.sync()
	return d.longest
}

func (d *Dict) Len() int {
	if d == nil {
		return 0
	}
	n := 0
	for _, list := range d.entries {
		n += len(list)
	}
	return n
}

func (d *Dict) Codes() []string {
	if d == nil {
		return nil
	}
	d.sync()
	return d.codes
}

func (d *Dict) Prefixed(prefix string, limit int) []string {
	if d == nil {
		return nil
	}
	d.sync()
	i := sort.SearchStrings(d.codes, prefix)
	var out []string
	for ; i < len(d.codes) && strings.HasPrefix(d.codes[i], prefix); i++ {
		if limit > 0 && len(out) == limit {
			break
		}
		out = append(out, d.codes[i])
	}
	return out
}

func (d *Dict) HasPrefix(prefix string) bool {
	if d == nil {
		return false
	}
	d.sync()
	i := sort.SearchStrings(d.codes, prefix)
	return i < len(d.codes) && strings.HasPrefix(d.codes[i], prefix)
}

func (d *Dict) Each(fn func(code string, entries []Entry)) {
	if d == nil {
		return
	}
	d.sync()
	for _, code := range d.order {
		fn(code, d.entries[code])
	}
}

func (d *Dict) Get(key string) string {
	if d == nil {
		return ""
	}
	return d.Meta[key]
}

func (d *Dict) Int(key string) int {
	n, _ := strconv.Atoi(d.Get(key))
	return n
}

var headerOrder = []string{"format", "name", "type", "language", "keys", "length", "wildcard", "autocommit", "license", "source"}

func (d *Dict) Write(w io.Writer) error {
	d.sync()
	b := bufio.NewWriter(w)
	meta := map[string]string{}
	for k, v := range d.Meta {
		meta[k] = v
	}
	meta["format"] = Format
	var keys []string
	for _, k := range headerOrder {
		if _, ok := meta[k]; ok {
			keys = append(keys, k)
		}
	}
	var rest []string
	for k := range meta {
		if !contains(headerOrder, k) {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range append(keys, rest...) {
		if strings.ContainsAny(meta[k], "\n") {
			return fmt.Errorf("header %s spans lines", k)
		}
		fmt.Fprintf(b, "@%s %s\n", k, meta[k])
	}
	for _, code := range d.order {
		for _, e := range d.entries[code] {
			if strings.ContainsAny(e.Code+e.Text, "\t\n") {
				continue
			}
			if e.Weight != 0 {
				fmt.Fprintf(b, "%s\t%s\t%d\n", e.Code, e.Text, e.Weight)
			} else {
				fmt.Fprintf(b, "%s\t%s\n", e.Code, e.Text)
			}
		}
	}
	return b.Flush()
}

func (d *Dict) Save(path string) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	var w io.Writer = f
	var z *gzip.Writer
	if strings.HasSuffix(path, ".gz") {
		z, _ = gzip.NewWriterLevel(f, gzip.BestCompression)
		w = z
	}
	err = d.Write(w)
	if z != nil && err == nil {
		err = z.Close()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
