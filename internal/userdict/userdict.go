package userdict

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/singularityos-lab/sinty-input/internal/dict"
)

var Delay = 2 * time.Second

type Store struct {
	mu    sync.Mutex
	name  string
	d     *dict.Dict
	timer *time.Timer
}

var (
	mu     sync.Mutex
	stores = map[string]*Store{}
)

func Dir() string {
	return filepath.Join(dict.UserDir(), "user")
}

func Open(name string) *Store {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := stores[name]; ok {
		return s
	}
	s := &Store{name: name}
	stores[name] = s
	return s
}

func Names() []string {
	entries, _ := os.ReadDir(Dir())
	var out []string
	for _, e := range entries {
		if n := e.Name(); filepath.Ext(n) == dict.Ext {
			out = append(out, dict.NameOf(n))
		}
	}
	sort.Strings(out)
	return out
}

func (s *Store) Path() string {
	return filepath.Join(Dir(), s.name+dict.Ext)
}

func (s *Store) load() *dict.Dict {
	if s.d != nil {
		return s.d
	}
	d, err := dict.Load(s.Path())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Print(err)
		}
		d = dict.New(s.name, nil)
	}
	d.Meta["name"] = s.name
	d.Meta["type"] = "user"
	d.Meta["license"] = "LicenseRef-personal"
	s.d = d
	return d
}

func (s *Store) Weight(code, text string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, _ := s.load().Weight(code, text)
	return w
}

func (s *Store) Lookup(code string) []dict.Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]dict.Entry(nil), s.load().Lookup(code)...)
}

func (s *Store) HasPrefix(prefix string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load().HasPrefix(prefix)
}

func (s *Store) Learn(code, text string) {
	if code == "" || text == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.load()
	w, _ := d.Weight(code, text)
	d.Set(code, text, w+1)
	s.schedule()
}

func (s *Store) Set(code, text string, weight int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load().Set(code, text, weight)
	s.schedule()
}

func (s *Store) Remove(code, text string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ok := s.load().Remove(code, text)
	if ok {
		s.schedule()
	}
	return ok
}

func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.d = dict.New(s.name, nil)
	s.load()
	err := os.Remove(s.Path())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Store) Entries() []dict.Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []dict.Entry
	s.load().Each(func(_ string, list []dict.Entry) {
		out = append(out, list...)
	})
	return out
}

func (s *Store) Export(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load().Save(path)
}

func (s *Store) Import(path string) (int, error) {
	in, err := dict.Load(path)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.load()
	n := 0
	in.Each(func(code string, list []dict.Entry) {
		for _, e := range list {
			w, _ := d.Weight(code, e.Text)
			d.Set(code, e.Text, max(w, e.Weight, 1))
			n++
		}
	})
	if n > 0 {
		s.schedule()
	}
	return n, nil
}

func (s *Store) schedule() {
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(Delay, func() {
		if err := s.Flush(); err != nil {
			log.Print(err)
		}
	})
}

func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.d == nil {
		return nil
	}
	if s.d.Len() == 0 {
		if err := os.Remove(s.Path()); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	return s.d.Save(s.Path())
}

func FlushAll() {
	mu.Lock()
	list := make([]*Store, 0, len(stores))
	for _, s := range stores {
		list = append(list, s)
	}
	mu.Unlock()
	for _, s := range list {
		if err := s.Flush(); err != nil {
			log.Print(err)
		}
	}
}

func Forget() {
	mu.Lock()
	defer mu.Unlock()
	for _, s := range stores {
		s.mu.Lock()
		if s.timer != nil {
			s.timer.Stop()
		}
		s.mu.Unlock()
	}
	stores = map[string]*Store{}
}

func Rank(code string, system []dict.Entry, user *Store) []string {
	var learned []dict.Entry
	if user != nil {
		learned = user.Lookup(code)
	}
	seen := map[string]bool{}
	var out []string
	for _, e := range learned {
		if !seen[e.Text] {
			seen[e.Text] = true
			out = append(out, e.Text)
		}
	}
	for _, e := range system {
		if !seen[e.Text] {
			seen[e.Text] = true
			out = append(out, e.Text)
		}
	}
	return out
}
