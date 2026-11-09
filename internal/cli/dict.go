package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	d, err := importer.Import(data, importer.Options{
		Format: *format, Name: *name, Type: *kind, Language: *language, License: *license,
		Source: *source, Keys: *keys, Length: *length, Wildcard: *wildcard, Fallback: dict.NameOf(out),
	})
	if err != nil {
		return err
	}
	if err := d.Save(out); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "%s: %d entries\n", out, d.Len())
	return nil
}
