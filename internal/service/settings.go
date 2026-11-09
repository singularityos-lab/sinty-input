package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/godbus/dbus/v5"

	"github.com/singularityos-lab/sinty-input/internal/config"
	"github.com/singularityos-lab/sinty-input/internal/convert"
	"github.com/singularityos-lab/sinty-input/internal/dict"
	"github.com/singularityos-lab/sinty-input/internal/dict/importer"
	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

type kinded interface {
	Kind() string
	Dictionary() string
}

type Method struct {
	Name     string
	Label    string
	Symbol   string
	Language string
	Options  map[string]dbus.Variant
}

func (s *Service) find(name string) engine.Engine {
	for _, e := range s.engines {
		if e.Info().Name == name {
			return e
		}
	}
	return nil
}

func chinese(language string) bool {
	return strings.HasPrefix(language, "zh") || language == "yue"
}

func (s *Service) describe(e engine.Engine) Method {
	i := e.Info()
	opts := map[string]dbus.Variant{}
	kind := "other"
	if k, ok := e.(kinded); ok {
		kind = k.Kind()
		opts["dictionary"] = dbus.MakeVariant(k.Dictionary())
		var choices []string
		for _, info := range dict.Catalog() {
			if info.Meta["type"] == kind && kind == "table" && chinese(info.Meta["language"]) == chinese(i.Language) {
				choices = append(choices, info.Name)
			}
		}
		if choices == nil {
			choices = []string{k.Dictionary()}
		}
		opts["dictionaries"] = dbus.MakeVariant(choices)
	}
	opts["kind"] = dbus.MakeVariant(kind)
	if c, ok := e.(engine.Configurable); ok {
		o := c.Options()
		conv := o.Conversion
		if conv == "" {
			conv = "none"
		}
		opts["fullwidth"] = dbus.MakeVariant(o.FullWidth)
		opts["learn"] = dbus.MakeVariant(o.Learn)
		if chinese(i.Language) {
			opts["conversion"] = dbus.MakeVariant(conv)
			opts["conversions"] = dbus.MakeVariant([]string{"none", convert.Simplified, convert.Traditional, convert.Taiwan, convert.HongKong})
		}
		opts["learned"] = dbus.MakeVariant(uint32(len(userdict.Open(i.Name).Entries())))
	}
	return Method{i.Name, i.Label, i.Symbol, i.Language, opts}
}

func (s *Service) Methods() ([]Method, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Method{}
	for _, e := range s.engines {
		out = append(out, s.describe(e))
	}
	return out, nil
}

func invalid(what ...interface{}) *dbus.Error {
	return dbus.NewError(Iface+".Error.InvalidArgs", what)
}

func (s *Service) Configure(name string, options map[string]dbus.Variant) *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.find(name)
	if e == nil {
		return dbus.NewError(Iface+".Error.UnknownEngine", []interface{}{name})
	}
	c, ok := e.(engine.Configurable)
	if !ok {
		return invalid(name)
	}
	cfg := config.Load()
	m := cfg.Methods[name]
	for k, v := range options {
		switch k {
		case "conversion":
			mode, ok := v.Value().(string)
			if !ok || !validConversion(mode) {
				return invalid("conversion")
			}
			m.Conversion = mode
		case "fullwidth":
			b, ok := v.Value().(bool)
			if !ok {
				return invalid("fullwidth")
			}
			m.FullWidth = config.Bool(b)
		case "learn":
			b, ok := v.Value().(bool)
			if !ok {
				return invalid("learn")
			}
			m.Learn = config.Bool(b)
		case "dictionary":
			d, ok := v.Value().(string)
			if !ok {
				return invalid("dictionary")
			}
			if _, found := dict.Find(d); d != "" && !found {
				return invalid("dictionary", d)
			}
			m.Dictionary = d
		default:
			return invalid("option", k)
		}
	}
	cfg.Methods[name] = m
	if err := cfg.Save(); err != nil {
		return dbus.MakeFailedError(err)
	}
	if s.active == e {
		e.Flush(s.out())
	}
	o := c.Options()
	if m.Conversion != "" {
		o.Conversion = m.Conversion
	}
	o.FullWidth, o.Learn, o.Dictionary = m.Wide(), m.Learning(), m.Dictionary
	c.Configure(o)
	return nil
}

func validConversion(mode string) bool {
	for _, m := range []string{"none", convert.Simplified, convert.Traditional, convert.Taiwan, convert.HongKong} {
		if mode == m {
			return true
		}
	}
	return false
}

type Dictionary struct {
	Name     string
	Type     string
	Language string
	Title    string
	License  string
	User     bool
}

func (s *Service) Dictionaries() ([]Dictionary, *dbus.Error) {
	out := []Dictionary{}
	for _, info := range dict.Catalog() {
		out = append(out, Dictionary{info.Name, info.Meta["type"], info.Meta["language"], info.Meta["name"], info.Meta["license"], info.User})
	}
	return out, nil
}

type Learned struct {
	Code   string
	Text   string
	Weight int32
}

func (s *Service) store(name string) (*userdict.Store, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.find(name) == nil {
		return nil, dbus.NewError(Iface+".Error.UnknownEngine", []interface{}{name})
	}
	return userdict.Open(name), nil
}

func (s *Service) UserEntries(name string) ([]Learned, *dbus.Error) {
	st, derr := s.store(name)
	if derr != nil {
		return nil, derr
	}
	out := []Learned{}
	for _, e := range st.Entries() {
		out = append(out, Learned{e.Code, e.Text, int32(e.Weight)})
	}
	return out, nil
}

func (s *Service) UserSet(name, code, text string, weight int32) *dbus.Error {
	st, derr := s.store(name)
	if derr != nil {
		return derr
	}
	if code == "" || text == "" || strings.ContainsAny(code+text, "\t\n") || weight < 0 {
		return invalid("entry")
	}
	st.Set(code, text, int(weight))
	return nil
}

func (s *Service) UserRemove(name, code, text string) *dbus.Error {
	st, derr := s.store(name)
	if derr != nil {
		return derr
	}
	st.Remove(code, text)
	return nil
}

func (s *Service) UserClear(name string) *dbus.Error {
	st, derr := s.store(name)
	if derr != nil {
		return derr
	}
	if err := st.Clear(); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func (s *Service) UserExport(name, path string) *dbus.Error {
	st, derr := s.store(name)
	if derr != nil {
		return derr
	}
	if !strings.HasSuffix(path, dict.Ext) && !strings.HasSuffix(path, dict.ExtGz) {
		return invalid("path")
	}
	if err := st.Export(path); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func (s *Service) UserImport(name, path string) (uint32, *dbus.Error) {
	st, derr := s.store(name)
	if derr != nil {
		return 0, derr
	}
	n, err := st.Import(path)
	if err != nil {
		return 0, dbus.MakeFailedError(err)
	}
	return uint32(n), nil
}

func (s *Service) Reload() *dbus.Error {
	if s.Rebuild == nil {
		return nil
	}
	engines := s.Rebuild()
	s.mu.Lock()
	defer s.mu.Unlock()
	active := ""
	if s.active != nil {
		active = s.active.Info().Name
		s.active.Flush(s.out())
	}
	s.engines = engines
	s.active = s.find(active)
	return nil
}

func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func (s *Service) AddDictionary(path, name, language, license string) (string, *dbus.Error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", dbus.MakeFailedError(err)
	}
	base := slug(name)
	if base == "" {
		base = slug(dict.NameOf(path))
	}
	if base == "" {
		base = "table"
	}
	d, err := importer.Import(data, importer.Options{Name: name, Language: language, License: license, Fallback: dict.NameOf(path)})
	if err != nil {
		return "", dbus.NewError(Iface+".Error.Import", []interface{}{err.Error()})
	}
	if d.Meta["type"] != "table" {
		return "", invalid("type", d.Meta["type"])
	}
	dir := filepath.Join(dict.UserDir(), "dict")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", dbus.MakeFailedError(err)
	}
	id := base
	for i := 2; ; i++ {
		if _, taken := dict.Find(id); !taken {
			break
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
	if err := d.Save(filepath.Join(dir, id+dict.ExtGz)); err != nil {
		return "", dbus.MakeFailedError(err)
	}
	dict.Forget(id)
	s.Reload()
	return "table-" + id, nil
}

func (s *Service) RemoveDictionary(name string) *dbus.Error {
	for _, info := range dict.Catalog() {
		if info.Name != name {
			continue
		}
		if !info.User {
			return invalid("system dictionary", name)
		}
		if err := os.Remove(info.Path); err != nil {
			return dbus.MakeFailedError(err)
		}
		dict.Forget(name)
		s.Reload()
		return nil
	}
	return invalid("dictionary", name)
}
