package ibus

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/singularityos-lab/sinty-input/internal/engine"
)

const (
	Component    = "dev.sinty.Input.IBus"
	FactoryPath  = dbus.ObjectPath("/org/freedesktop/IBus/Factory")
	FactoryIface = "org.freedesktop.IBus.Factory"
	EngineIface  = "org.freedesktop.IBus.Engine"
	ServiceIface = "org.freedesktop.IBus.Service"
	Prefix       = "sinty-"
	releaseMask  = 1 << 30
)

type Text struct {
	Name        string
	Attachments map[string]dbus.Variant
	Text        string
	Attributes  dbus.Variant
}

type AttrList struct {
	Name        string
	Attachments map[string]dbus.Variant
	Attributes  []dbus.Variant
}

type Attribute struct {
	Name        string
	Attachments map[string]dbus.Variant
	Type        uint32
	Value       uint32
	Start       uint32
	End         uint32
}

type LookupTable struct {
	Name          string
	Attachments   map[string]dbus.Variant
	PageSize      uint32
	Cursor        uint32
	CursorVisible bool
	Round         bool
	Orientation   int32
	Candidates    []dbus.Variant
	Labels        []dbus.Variant
}

func NewText(s string, underline bool) dbus.Variant {
	attrs := []dbus.Variant{}
	if underline && s != "" {
		attrs = append(attrs, dbus.MakeVariant(Attribute{"IBusAttribute", map[string]dbus.Variant{}, 1, 1, 0, uint32(engine.Runes(s))}))
	}
	return dbus.MakeVariant(Text{"IBusText", map[string]dbus.Variant{}, s, dbus.MakeVariant(AttrList{"IBusAttrList", map[string]dbus.Variant{}, attrs})})
}

func Language(tag string) string {
	switch {
	case tag == "zh-Hant":
		return "zh_TW"
	case tag == "yue":
		return "zh_HK"
	case strings.HasPrefix(tag, "zh"):
		return "zh_CN"
	}
	return strings.ReplaceAll(tag, "-", "_")
}

func EnginesXML(w io.Writer, engines []engine.Engine) {
	fmt.Fprintln(w, "<engines>")
	for _, e := range engines {
		i := e.Info()
		fmt.Fprintf(w, "  <engine>\n    <name>%s</name>\n    <longname>%s</longname>\n    <description>%s</description>\n    <language>%s</language>\n    <license>GPL-3.0-only</license>\n    <author>Sinty OS</author>\n    <layout>default</layout>\n    <symbol>%s</symbol>\n    <rank>0</rank>\n    <textdomain>sinty-input</textdomain>\n  </engine>\n",
			esc(Prefix+i.Name), esc(i.Label), esc(i.Label), esc(Language(i.Language)), esc(i.Symbol))
	}
	fmt.Fprintln(w, "</engines>")
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func Address() (string, error) {
	if a := os.Getenv("IBUS_ADDRESS"); a != "" {
		return a, nil
	}
	path := os.Getenv("IBUS_ADDRESS_FILE")
	if path == "" {
		id, err := os.ReadFile("/var/lib/dbus/machine-id")
		if err != nil {
			id, err = os.ReadFile("/etc/machine-id")
		}
		if err != nil {
			return "", err
		}
		display := "0"
		host := "unix"
		if d := os.Getenv("WAYLAND_DISPLAY"); d != "" {
			display = d
		} else if d := os.Getenv("DISPLAY"); d != "" {
			h, rest, _ := strings.Cut(d, ":")
			if h != "" {
				host = h
			}
			display, _, _ = strings.Cut(rest, ".")
		}
		dir := os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			home, _ := os.UserHomeDir()
			dir = filepath.Join(home, ".config")
		}
		path = filepath.Join(dir, "ibus", "bus", fmt.Sprintf("%s-%s-%s", strings.TrimSpace(string(id)), host, display))
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if v, ok := strings.CutPrefix(s.Text(), "IBUS_ADDRESS="); ok {
			return v, nil
		}
	}
	return "", errors.New("no IBUS_ADDRESS in " + path)
}

type Factory struct {
	conn  *dbus.Conn
	make  func(name string) engine.Engine
	mu    sync.Mutex
	next  int
	alive map[dbus.ObjectPath]*Engine
}

func Serve(conn *dbus.Conn, make func(name string) engine.Engine) (*Factory, error) {
	f := &Factory{conn: conn, make: make, alive: map[dbus.ObjectPath]*Engine{}}
	if err := conn.Export(f, FactoryPath, FactoryIface); err != nil {
		return nil, err
	}
	node := &introspect.Node{Name: string(FactoryPath), Interfaces: []introspect.Interface{introspect.IntrospectData, {
		Name: FactoryIface, Methods: introspect.Methods(f),
	}}}
	if err := conn.Export(introspect.NewIntrospectable(node), FactoryPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, err
	}
	reply, err := conn.RequestName(Component, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, errors.New(Component + " is taken")
	}
	return f, nil
}

func (f *Factory) CreateEngine(name string) (dbus.ObjectPath, *dbus.Error) {
	e := f.make(strings.TrimPrefix(name, Prefix))
	if e == nil {
		return "", dbus.NewError("org.freedesktop.IBus.Error.Failed", []interface{}{"unknown engine " + name})
	}
	f.mu.Lock()
	f.next++
	path := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/IBus/Engine/%d", f.next))
	f.mu.Unlock()
	ie := &Engine{conn: f.conn, path: path, engine: e, factory: f}
	if err := ie.export(); err != nil {
		return "", dbus.MakeFailedError(err)
	}
	f.mu.Lock()
	f.alive[path] = ie
	f.mu.Unlock()
	return path, nil
}

type Engine struct {
	mu      sync.Mutex
	conn    *dbus.Conn
	path    dbus.ObjectPath
	engine  engine.Engine
	factory *Factory
	enabled bool
}

type service struct{ e *Engine }

func (s service) Destroy() *dbus.Error {
	s.e.destroy()
	return nil
}

func (e *Engine) export() error {
	if err := e.conn.Export(e, e.path, EngineIface); err != nil {
		return err
	}
	if err := e.conn.Export(service{e}, e.path, ServiceIface); err != nil {
		return err
	}
	props, err := prop.Export(e.conn, e.path, prop.Map{EngineIface: {
		"FocusId":               {Value: false},
		"ActiveSurroundingText": {Value: false},
		"ContentType":           {Value: struct{ Purpose, Hints uint32 }{}, Writable: true},
	}})
	if err != nil {
		return err
	}
	node := &introspect.Node{Name: string(e.path), Interfaces: []introspect.Interface{introspect.IntrospectData, prop.IntrospectData,
		{Name: EngineIface, Methods: introspect.Methods(e), Properties: props.Introspection(EngineIface), Signals: signals},
		{Name: ServiceIface, Methods: introspect.Methods(service{e})},
	}}
	return e.conn.Export(introspect.NewIntrospectable(node), e.path, "org.freedesktop.DBus.Introspectable")
}

var signals = []introspect.Signal{
	{Name: "CommitText", Args: []introspect.Arg{{Name: "text", Type: "v"}}},
	{Name: "UpdatePreeditText", Args: []introspect.Arg{{Name: "text", Type: "v"}, {Name: "cursor_pos", Type: "u"}, {Name: "visible", Type: "b"}, {Name: "mode", Type: "u"}}},
	{Name: "HidePreeditText"},
	{Name: "UpdateLookupTable", Args: []introspect.Arg{{Name: "table", Type: "v"}, {Name: "visible", Type: "b"}}},
	{Name: "HideLookupTable"},
	{Name: "UpdateAuxiliaryText", Args: []introspect.Arg{{Name: "text", Type: "v"}, {Name: "visible", Type: "b"}}},
	{Name: "HideAuxiliaryText"},
}

func (e *Engine) destroy() {
	e.mu.Lock()
	e.engine.Reset(discard{})
	e.mu.Unlock()
	for _, iface := range []string{EngineIface, ServiceIface, "org.freedesktop.DBus.Introspectable", "org.freedesktop.DBus.Properties"} {
		e.conn.Export(nil, e.path, iface)
	}
	e.factory.mu.Lock()
	delete(e.factory.alive, e.path)
	e.factory.mu.Unlock()
}

type discard struct{}

func (discard) Commit(string)                 {}
func (discard) Preedit(string, int)           {}
func (discard) Candidates(*engine.Candidates) {}

type output struct{ e *Engine }

func (o output) emit(name string, values ...interface{}) {
	o.e.conn.Emit(o.e.path, EngineIface+"."+name, values...)
}

func (o output) Commit(text string) {
	o.emit("CommitText", NewText(text, false))
}

func (o output) Preedit(text string, cursor int) {
	if text == "" {
		o.emit("UpdatePreeditText", NewText("", false), uint32(0), false, uint32(0))
		o.emit("HidePreeditText")
		return
	}
	o.emit("UpdatePreeditText", NewText(text, true), uint32(cursor), true, uint32(0))
}

func (o output) Candidates(c *engine.Candidates) {
	if c == nil || len(c.Items) == 0 {
		o.emit("HideLookupTable")
		o.emit("HideAuxiliaryText")
		return
	}
	t := LookupTable{Name: "IBusLookupTable", Attachments: map[string]dbus.Variant{}, PageSize: uint32(len(c.Items)),
		Cursor: uint32(c.Selected), CursorVisible: true, Orientation: 2, Candidates: []dbus.Variant{}, Labels: []dbus.Variant{}}
	for i, item := range c.Items {
		t.Candidates = append(t.Candidates, NewText(item, false))
		label := ""
		if i < len(c.Labels) {
			label = c.Labels[i]
		}
		t.Labels = append(t.Labels, NewText(label, false))
	}
	o.emit("UpdateLookupTable", dbus.MakeVariant(t), true)
	if c.Aux != "" {
		o.emit("UpdateAuxiliaryText", NewText(c.Aux, false), true)
	} else {
		o.emit("HideAuxiliaryText")
	}
}

func (e *Engine) out() engine.Output {
	return output{e}
}

func (e *Engine) ProcessKeyEvent(keyval, keycode, state uint32) (bool, *dbus.Error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	k := engine.Key{Sym: keyval, Code: keycode, State: state &^ releaseMask, Release: state&releaseMask != 0}
	return e.engine.Key(k, e.out()), nil
}

func (e *Engine) CandidateClicked(index, button, state uint32) *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine.Select(int(index), e.out())
	return nil
}

func (e *Engine) PageUp() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine.Page(false, e.out())
	return nil
}

func (e *Engine) PageDown() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine.Page(true, e.out())
	return nil
}

func (e *Engine) CursorUp() *dbus.Error {
	_, err := e.ProcessKeyEvent(engine.KeyUp, 0, 0)
	return err
}

func (e *Engine) CursorDown() *dbus.Error {
	_, err := e.ProcessKeyEvent(engine.KeyDown, 0, 0)
	return err
}

func (e *Engine) FocusIn() *dbus.Error {
	return nil
}

func (e *Engine) FocusInId(object, client string) *dbus.Error {
	return nil
}

func (e *Engine) FocusOut() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine.Flush(e.out())
	return nil
}

func (e *Engine) FocusOutId(object string) *dbus.Error {
	return e.FocusOut()
}

func (e *Engine) Reset() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engine.Reset(e.out())
	return nil
}

func (e *Engine) Enable() *dbus.Error {
	e.mu.Lock()
	e.enabled = true
	e.mu.Unlock()
	return nil
}

func (e *Engine) Disable() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = false
	e.engine.Flush(e.out())
	return nil
}

func (e *Engine) SetCapabilities(caps uint32) *dbus.Error {
	return nil
}

func (e *Engine) SetCursorLocation(x, y, w, h int32) *dbus.Error {
	return nil
}

func (e *Engine) SetCursorLocationRelative(x, y, w, h int32) *dbus.Error {
	return nil
}

func (e *Engine) SetSurroundingText(text dbus.Variant, cursor, anchor uint32) *dbus.Error {
	return nil
}

func (e *Engine) PropertyActivate(name string, state uint32) *dbus.Error {
	return nil
}

func (e *Engine) PropertyShow(name string) *dbus.Error {
	return nil
}

func (e *Engine) PropertyHide(name string) *dbus.Error {
	return nil
}

func (e *Engine) ProcessHandWritingEvent(coordinates []float64) *dbus.Error {
	return nil
}

func (e *Engine) CancelHandWriting(strokes uint32) *dbus.Error {
	return nil
}

func (e *Engine) PanelExtensionReceived(event dbus.Variant) *dbus.Error {
	return nil
}

func (e *Engine) PanelExtensionRegisterKeys(data dbus.Variant) *dbus.Error {
	return nil
}
