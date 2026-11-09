package dict

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	Ext   = ".sdict"
	ExtGz = ".sdict.gz"
)

func NameOf(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".gz")
	return strings.TrimSuffix(base, Ext)
}

func dataHome() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

func UserDir() string {
	if d := os.Getenv("SINTY_INPUT_USER_DIR"); d != "" {
		return d
	}
	return filepath.Join(dataHome(), "sinty-input")
}

func Dirs() []string {
	if p := os.Getenv("SINTY_INPUT_DICT_PATH"); p != "" {
		return filepath.SplitList(p)
	}
	dirs := []string{filepath.Join(UserDir(), "dict")}
	system := os.Getenv("XDG_DATA_DIRS")
	if system == "" {
		system = "/usr/local/share:/usr/share"
	}
	for _, d := range filepath.SplitList(system) {
		if d != "" {
			dirs = append(dirs, filepath.Join(d, "sinty-input", "dict"))
		}
	}
	return dirs
}

type Info struct {
	Name string
	Path string
	Meta map[string]string
	User bool
}

func Find(name string) (string, bool) {
	for _, dir := range Dirs() {
		for _, ext := range []string{Ext, ExtGz} {
			p := filepath.Join(dir, name+ext)
			if _, err := os.Stat(p); err == nil {
				return p, true
			}
		}
	}
	return "", false
}

func Catalog() []Info {
	seen := map[string]bool{}
	var out []Info
	user := filepath.Join(UserDir(), "dict")
	for _, dir := range Dirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			n := e.Name()
			if e.IsDir() || !(strings.HasSuffix(n, Ext) || strings.HasSuffix(n, ExtGz)) {
				continue
			}
			name := NameOf(n)
			if seen[name] {
				continue
			}
			p := filepath.Join(dir, n)
			meta, err := ReadHeader(p)
			if err != nil {
				continue
			}
			seen[name] = true
			out = append(out, Info{name, p, meta, dir == user})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

var (
	mu     sync.Mutex
	loaded = map[string]*Dict{}
)

func Get(name string) *Dict {
	mu.Lock()
	defer mu.Unlock()
	if d, ok := loaded[name]; ok {
		return d
	}
	var d *Dict
	if p, ok := Find(name); ok {
		var err error
		if d, err = Load(p); err != nil {
			log.Print(err)
			d = nil
		}
	}
	loaded[name] = d
	return d
}

func Forget(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(loaded, name)
}
