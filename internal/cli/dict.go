package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/dict/importer"
)

func Dict(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sinty-input dict list|import")
		return 2
	}
	var err error
	switch args[0] {
	case "list":
		err = list(stdout)
	case "import":
		err = importFile(args[1:], stderr)
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil {
		fmt.Fprintln(stderr, "sinty-input:", err)
		return 1
	}
	return 0
}

func list(w io.Writer) error {
	for _, info := range dict.Catalog() {
		where := "system"
		if info.User {
			where = "user"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", info.Name, info.Meta["type"], info.Meta["language"], info.Meta["name"], info.Meta["license"], where)
	}
	return nil
}

func importFile(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "", "rime, fcitx or sinty; detected when empty")
	license := fs.String("license", "", "SPDX identifier of the license of the input")
	name := fs.String("name", "", "name shown in Settings")
	kind := fs.String("type", "table", "table, phonetic or conversion")
	language := fs.String("language", "", "BCP 47 language tag")
	source := fs.String("source", "", "where the data comes from")
	keys := fs.String("keys", "", "keys that make up codes")
	length := fs.String("length", "", "longest code")
	wildcard := fs.String("wildcard", "", "key that matches any one key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: sinty-input dict import [flags] INPUT OUTPUT.sdict[.gz]")
	}
	in, out := fs.Arg(0), fs.Arg(1)
	if !strings.HasSuffix(out, dict.Ext) && !strings.HasSuffix(out, dict.ExtGz) {
		return fmt.Errorf("%s must end in %s or %s", out, dict.Ext, dict.ExtGz)
	}
	if !importer.Compatible(*license) {
		known := importer.Licenses()
		sort.Strings(known)
		return fmt.Errorf("the license %q is not one that can be imported; give -license with one of: %s", *license, strings.Join(known, ", "))
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	if *format == "" {
		*format = importer.Detect(data[:min(len(data), 4096)])
	}
	var d *dict.Dict
	r := strings.NewReader(string(data))
	switch *format {
	case "rime":
		d, err = importer.Rime(r)
	case "fcitx":
		d, err = importer.Fcitx(r)
	case "sinty":
		d, err = dict.Read(r)
	default:
		return errors.New("unknown input format; give -format rime or -format fcitx")
	}
	if err != nil {
		return err
	}
	delete(d.Meta, "rime")
	set := func(k, v string) {
		if v != "" {
			d.Meta[k] = v
		}
	}
	set("name", *name)
	set("type", *kind)
	set("language", *language)
	set("license", *license)
	set("source", *source)
	set("keys", *keys)
	set("length", *length)
	set("wildcard", *wildcard)
	if d.Meta["name"] == "" {
		d.Meta["name"] = dict.NameOf(out)
	}
	if d.Meta["type"] == "table" && d.Meta["keys"] == "" {
		d.Meta["keys"] = keysOf(d)
	}
	if d.Meta["type"] == "table" && d.Meta["length"] == "" {
		d.Meta["length"] = fmt.Sprint(d.Longest())
	}
	if d.Len() == 0 {
		return errors.New("no entries found")
	}
	if err := d.Save(out); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "%s: %d entries\n", out, d.Len())
	return nil
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
