package importer

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/singularityos-lab/sinty-input/internal/dict"
)

type Options struct {
	Format   string
	Name     string
	Type     string
	Language string
	License  string
	Source   string
	Keys     string
	Length   string
	Wildcard string
	Fallback string
}

func Import(data []byte, o Options) (*dict.Dict, error) {
	format := o.Format
	if format == "" {
		format = Detect(data[:min(len(data), 4096)])
	}
	var d *dict.Dict
	var err error
	r := strings.NewReader(string(data))
	switch format {
	case "rime":
		d, err = Rime(r)
	case "fcitx":
		d, err = Fcitx(r)
	case "sinty":
		d, err = dict.Read(r)
	default:
		return nil, errors.New("unknown input format; give -format rime or -format fcitx")
	}
	if err != nil {
		return nil, err
	}
	delete(d.Meta, "rime")
	set := func(k, v string) {
		if v != "" {
			d.Meta[k] = v
		}
	}
	set("name", o.Name)
	set("type", o.Type)
	set("language", o.Language)
	set("license", o.License)
	set("source", o.Source)
	set("keys", o.Keys)
	set("length", o.Length)
	set("wildcard", o.Wildcard)
	if d.Meta["type"] == "" {
		d.Meta["type"] = "table"
	}
	if !Compatible(d.Meta["license"]) {
		known := Licenses()
		sort.Strings(known)
		return nil, fmt.Errorf("the license %q is not one that can be imported; use one of: %s", d.Meta["license"], strings.Join(known, ", "))
	}
	if d.Meta["name"] == "" {
		d.Meta["name"] = o.Fallback
	}
	if d.Meta["type"] == "table" && d.Meta["keys"] == "" {
		d.Meta["keys"] = keysOf(d)
	}
	if d.Meta["type"] == "table" && d.Meta["length"] == "" {
		d.Meta["length"] = fmt.Sprint(d.Longest())
	}
	if d.Len() == 0 {
		return nil, errors.New("no entries found")
	}
	return d, nil
}

func keysOf(d *dict.Dict) string {
	seen := map[rune]bool{}
	for _, c := range d.Codes() {
		for _, r := range c {
			seen[r] = true
		}
	}
	var out []rune
	for r := range seen {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return string(out)
}
