package service

import (
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/singularityos-lab/sinty-input/internal/engine"
)

const (
	Name  = "dev.sinty.Input1"
	Path  = dbus.ObjectPath("/dev/sinty/Input1")
	Iface = "dev.sinty.Input1"
)

type Emitter interface {
	Emit(path dbus.ObjectPath, name string, values ...interface{}) error
}

type Service struct {
	mu      sync.Mutex
	emitter Emitter
	engines []engine.Engine
	active  engine.Engine
	props   *prop.Properties
}

type signals struct {
	emitter Emitter
}

func (s signals) Commit(text string) {
	s.emitter.Emit(Path, Iface+".Commit", text)
}

func (s signals) Preedit(text string, cursor int) {
	s.emitter.Emit(Path, Iface+".Preedit", text, int32(cursor))
}

func (s signals) Candidates(c *engine.Candidates) {
	if c == nil {
		c = &engine.Candidates{}
	}
	items, labels := c.Items, c.Labels
	if items == nil {
		items = []string{}
	}
	if labels == nil {
		labels = []string{}
	}
	s.emitter.Emit(Path, Iface+".Candidates", items, labels, int32(c.Selected), c.Previous, c.Next, c.Aux)
}

func NewService(emitter Emitter, engines []engine.Engine) *Service {
	return &Service{emitter: emitter, engines: engines}
}

func Export(conn *dbus.Conn, engines []engine.Engine) (*Service, error) {
	s := NewService(conn, engines)
	if err := conn.Export(s, Path, Iface); err != nil {
		return nil, err
	}
	props, err := prop.Export(conn, Path, prop.Map{Iface: {
		"Engine": {Value: "", Emit: prop.EmitTrue},
	}})
	if err != nil {
		return nil, err
	}
	s.props = props
	node := &introspect.Node{
		Name: string(Path),
		Interfaces: []introspect.Interface{introspect.IntrospectData, prop.IntrospectData, {
			Name:       Iface,
			Methods:    introspect.Methods(s),
			Properties: props.Introspection(Iface),
			Signals: []introspect.Signal{
				{Name: "Commit", Args: []introspect.Arg{{Name: "text", Type: "s"}}},
				{Name: "Preedit", Args: []introspect.Arg{{Name: "text", Type: "s"}, {Name: "cursor", Type: "i"}}},
				{Name: "Candidates", Args: []introspect.Arg{
					{Name: "items", Type: "as"}, {Name: "labels", Type: "as"}, {Name: "selected", Type: "i"},
					{Name: "previous", Type: "b"}, {Name: "next", Type: "b"}, {Name: "auxiliary", Type: "s"},
				}},
			},
		}},
	}
	if err := conn.Export(introspect.NewIntrospectable(node), Path, "org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) out() engine.Output {
	return signals{s.emitter}
}

func (s *Service) ListEngines() ([][]string, *dbus.Error) {
	out := [][]string{}
	for _, e := range s.engines {
		i := e.Info()
		out = append(out, []string{i.Name, i.Label, i.Symbol, i.Language})
	}
	return out, nil
}

func (s *Service) SelectEngine(name string) *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var next engine.Engine
	for _, e := range s.engines {
		if e.Info().Name == name {
			next = e
		}
	}
	if next == nil && name != "" {
		return dbus.NewError(Iface+".Error.UnknownEngine", []interface{}{name})
	}
	if s.active != nil && s.active != next {
		s.active.Flush(s.out())
	}
	s.active = next
	if s.props != nil {
		s.props.SetMust(Iface, "Engine", name)
	}
	return nil
}

func (s *Service) ProcessKey(keysym, keycode, state uint32, release bool) (bool, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		return false, nil
	}
	return s.active.Key(engine.Key{Sym: keysym, Code: keycode, State: state, Release: release}, s.out()), nil
}

func (s *Service) FocusIn() *dbus.Error {
	return nil
}

func (s *Service) FocusOut() *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		s.active.Flush(s.out())
	}
	return nil
}

func (s *Service) Reset() *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		s.active.Reset(s.out())
	}
	return nil
}

func (s *Service) SetSurrounding(text string, cursor uint32) *dbus.Error {
	return nil
}

func (s *Service) SelectCandidate(index int32) *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		s.active.Select(int(index), s.out())
	}
	return nil
}

func (s *Service) ChangePage(next bool) *dbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		s.active.Page(next, s.out())
	}
	return nil
}
