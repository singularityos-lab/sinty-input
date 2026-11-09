package service

import (
	_ "github.com/singularityos-lab/sinty-input/internal/dict/dicttest"
	"testing"

	"github.com/godbus/dbus/v5"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/hangul"
	"github.com/singularityos-lab/sinty-input/internal/telex"
)

type emitted struct {
	name   string
	values []interface{}
}

type recorder struct {
	signals []emitted
}

func (r *recorder) Emit(path dbus.ObjectPath, name string, values ...interface{}) error {
	r.signals = append(r.signals, emitted{name, values})
	return nil
}

func (r *recorder) commits() string {
	out := ""
	for _, s := range r.signals {
		if s.name == Iface+".Commit" {
			out += s.values[0].(string)
		}
	}
	return out
}

func TestNoEngineLetsKeysThrough(t *testing.T) {
	s := NewService(&recorder{}, []engine.Engine{hangul.New()})
	if handled, _ := s.ProcessKey('a', 38, 0, false); handled {
		t.Fatal("without an engine every key must reach the application")
	}
}

func TestSwitchingEngineCommitsComposition(t *testing.T) {
	r := &recorder{}
	s := NewService(r, []engine.Engine{hangul.New(), telex.New()})
	if err := s.SelectEngine("hangul"); err != nil {
		t.Fatal(err)
	}
	for _, c := range "gks" {
		if handled, _ := s.ProcessKey(uint32(c), 0, 0, false); !handled {
			t.Fatalf("%c not handled", c)
		}
	}
	if err := s.SelectEngine("telex"); err != nil {
		t.Fatal(err)
	}
	if r.commits() != "한" {
		t.Fatalf("committed %q", r.commits())
	}
	if err := s.SelectEngine("missing"); err == nil {
		t.Fatal("unknown engine accepted")
	}
}

func TestCandidatesSignalHasTypedArgs(t *testing.T) {
	r := &recorder{}
	signals{r}.Candidates(nil)
	v := r.signals[0].values
	if _, ok := v[0].([]string); !ok || len(v) != 6 {
		t.Fatalf("values %#v", v)
	}
}

func TestListEngines(t *testing.T) {
	s := NewService(&recorder{}, []engine.Engine{hangul.New()})
	list, _ := s.ListEngines()
	if len(list) != 1 || list[0][0] != "hangul" || list[0][3] != "ko" {
		t.Fatalf("%v", list)
	}
}
