package dict

import (
	_ "github.com/singularityos-lab/sinty-input/internal/dict/dicttest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShipped(t *testing.T) {
	cases := []struct {
		dict  string
		key   string
		first string
	}{
		{"ja-kana", "にほん", "日本"},
		{"ja-kana", "かk", "書"},
		{"zh-pinyin", "shi", "是"},
		{"zh-pinyin", "zhong guo", "中国"},
	}
	for _, c := range cases {
		got := Get(c.dict).Words(c.key)
		if len(got) == 0 || got[0] != c.first {
			t.Errorf("%s %q: got %v, want %q first", c.dict, c.key, got, c.first)
		}
	}
	if Get("ja-kana").Lookup("ゔゔゔゔ") != nil {
		t.Error("unknown reading must have no candidates")
	}
	if Get("missing") != nil {
		t.Error("a missing dictionary must be nil")
	}
	for _, info := range Catalog() {
		for _, k := range []string{"name", "type", "language", "license"} {
			if info.Meta[k] == "" {
				t.Errorf("%s has no @%s", info.Name, k)
			}
		}
	}
}

func TestFormat(t *testing.T) {
	src := "# test\n@format sinty-dict 1\n@name Test\n@type table\n\naa\t工\t5\naa\t式\naa\t工\t9\nab\t戈\t2\n"
	d, err := Read(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Words("aa"); len(got) != 2 || got[0] != "工" {
		t.Fatalf("aa: %v", got)
	}
	if w, _ := d.Weight("aa", "工"); w != 9 {
		t.Fatalf("duplicates keep the highest weight, got %d", w)
	}
	if got := d.Prefixed("a", 0); len(got) != 2 {
		t.Fatalf("prefixed: %v", got)
	}
	if !d.HasPrefix("ab") || d.HasPrefix("b") {
		t.Fatal("prefix test")
	}
	d.Set("ab", "戈", 20)
	d.Add("b", "子", 0)
	if !d.Remove("aa", "式") || d.Remove("aa", "式") {
		t.Fatal("remove")
	}
	path := filepath.Join(t.TempDir(), "user.sdict.gz")
	if err := d.Save(path); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Name != "user" || back.Get("name") != "Test" || back.Len() != 3 || back.Score("ab") != 20 {
		t.Fatalf("round trip: %s %v %d", back.Name, back.Meta, back.Len())
	}
	for _, bad := range []string{"aa\t工\n", "@format sinty-dict 1\naa\n", "@format sinty-dict 1\naa\t工\tx\n"} {
		if _, err := Read(strings.NewReader(bad)); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestSearchOrder(t *testing.T) {
	user, system := t.TempDir(), t.TempDir()
	t.Setenv("SINTY_INPUT_DICT_PATH", "")
	t.Setenv("SINTY_INPUT_USER_DIR", user)
	t.Setenv("XDG_DATA_DIRS", system)
	os.MkdirAll(filepath.Join(user, "dict"), 0o755)
	os.MkdirAll(filepath.Join(system, "sinty-input", "dict"), 0o755)
	write := func(dir, name, text string) {
		d := New(name, map[string]string{"name": text, "type": "table", "language": "zh", "license": "CC0-1.0"})
		d.Add("a", text, 0)
		if err := d.Save(filepath.Join(dir, name+Ext)); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(system, "sinty-input", "dict"), "same", "system")
	write(filepath.Join(user, "dict"), "same", "user")
	write(filepath.Join(system, "sinty-input", "dict"), "other", "system")
	Forget("same")
	defer Forget("same")
	if got := Get("same").Words("a"); len(got) != 1 || got[0] != "user" {
		t.Fatalf("user dictionaries must win: %v", got)
	}
	cat := Catalog()
	if len(cat) != 2 || !cat[1].User || cat[0].User {
		t.Fatalf("catalog: %+v", cat)
	}
}
