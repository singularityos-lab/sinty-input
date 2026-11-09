package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/singularityos-lab/sinty-input/internal/dict"
)

func TestImport(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "mine.txt")
	os.WriteFile(in, []byte("KeyCode=abc\nLength=2\n[Data]\na 工\nab 式\n"), 0o644)
	out := filepath.Join(dir, "mine.sdict.gz")
	var o, e bytes.Buffer
	if Dict([]string{"import", in, out}, &o, &e) == 0 {
		t.Fatal("imported without a license")
	}
	if !strings.Contains(e.String(), "LGPL-3.0-only") {
		t.Fatalf("the refusal must list the accepted licenses: %s", e.String())
	}
	if Dict([]string{"import", "-license", "LicenseRef-proprietary", in, out}, &o, &e) == 0 {
		t.Fatal("imported an incompatible license")
	}
	e.Reset()
	if code := Dict([]string{"import", "-license", "CC0-1.0", "-language", "zh-Hans", in, out}, &o, &e); code != 0 {
		t.Fatalf("import failed: %s", e.String())
	}
	d, err := dict.Load(out)
	if err != nil {
		t.Fatal(err)
	}
	if d.Words("ab")[0] != "式" || d.Get("license") != "CC0-1.0" || d.Get("name") != "mine" || d.Get("type") != "table" || d.Get("keys") != "abc" {
		t.Fatalf("imported %v", d.Meta)
	}
	t.Setenv("SINTY_INPUT_DICT_PATH", dir)
	o.Reset()
	if Dict([]string{"list"}, &o, &e) != 0 || !strings.HasPrefix(o.String(), "mine\ttable\tzh-Hans\tmine\tCC0-1.0\t") {
		t.Fatalf("list: %q", o.String())
	}
}
