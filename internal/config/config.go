package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Method struct {
	Conversion string `json:"conversion,omitempty"`
	FullWidth  *bool  `json:"fullwidth,omitempty"`
	Learn      *bool  `json:"learn,omitempty"`
	Dictionary string `json:"dictionary,omitempty"`
}

type Config struct {
	Methods map[string]Method `json:"methods"`
}

func Path() string {
	if p := os.Getenv("SINTY_INPUT_CONFIG"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "sinty-input", "config.json")
}

var mu sync.Mutex

func Load() Config {
	mu.Lock()
	defer mu.Unlock()
	c := Config{Methods: map[string]Method{}}
	data, err := os.ReadFile(Path())
	if err == nil {
		json.Unmarshal(data, &c)
	}
	if c.Methods == nil {
		c.Methods = map[string]Method{}
	}
	return c
}

func (c Config) Save() error {
	mu.Lock()
	defer mu.Unlock()
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (m Method) Wide() bool {
	return m.FullWidth == nil || *m.FullWidth
}

func (m Method) Learning() bool {
	return m.Learn == nil || *m.Learn
}

func Bool(v bool) *bool {
	return &v
}
