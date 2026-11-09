package userdict

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/singularityos-lab/sinty-input/internal/dict"
)

func TestLearnPersistsLocally(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SINTY_INPUT_USER_DIR", dir)
	Delay = 10 * time.Millisecond
	defer Forget()
	s := Open("pinyin")
	s.Learn("ni hao", "你好")
	s.Learn("ni hao", "你好")
	s.Learn("ni hao", "拟好")
	if s.Weight("ni hao", "你好") != 2 {
		t.Fatalf("weight %d", s.Weight("ni hao", "你好"))
	}
	system := []dict.Entry{{Code: "ni hao", Text: "拟好"}, {Code: "ni hao", Text: "妮好"}, {Code: "ni hao", Text: "你好"}}
	if got := Rank("ni hao", system, s); len(got) != 3 || got[0] != "你好" || got[1] != "拟好" || got[2] != "妮好" {
		t.Fatalf("rank %v", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "user", "pinyin.sdict")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("not saved")
		}
		time.Sleep(10 * time.Millisecond)
	}
	info, _ := os.Stat(filepath.Join(dir, "user"))
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("user directory mode %v", info.Mode().Perm())
	}
	Forget()
	back := Open("pinyin")
	if back.Weight("ni hao", "你好") != 2 || len(back.Entries()) != 2 {
		t.Fatalf("reload: %v", back.Entries())
	}
	out := filepath.Join(t.TempDir(), "export.sdict")
	if err := back.Export(out); err != nil {
		t.Fatal(err)
	}
	if !back.Remove("ni hao", "拟好") || len(back.Entries()) != 1 {
		t.Fatal("remove")
	}
	other := Open("wubi86")
	if n, err := other.Import(out); err != nil || n != 2 || other.Weight("ni hao", "你好") != 2 {
		t.Fatalf("import %d %v", n, err)
	}
	if err := back.Clear(); err != nil || len(back.Entries()) != 0 {
		t.Fatal("clear")
	}
	FlushAll()
	if got := Names(); len(got) != 1 || got[0] != "wubi86" {
		t.Fatalf("names %v", got)
	}
}
