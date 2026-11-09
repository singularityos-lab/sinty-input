package ibus

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"

	"github.com/singularityos-lab/sinty-input/internal/engine"
	"github.com/singularityos-lab/sinty-input/internal/hangul"
)

func TestSignatures(t *testing.T) {
	if got := NewText("你好", true).Signature().String(); got != "(sa{sv}sv)" {
		t.Fatalf("IBusText %s", got)
	}
	attrs := NewText("你好", true).Value().(Text).Attributes.Value().(AttrList)
	if len(attrs.Attributes) != 1 || attrs.Attributes[0].Signature().String() != "(sa{sv}uuuu)" {
		t.Fatalf("attributes %+v", attrs)
	}
	if a := attrs.Attributes[0].Value().(Attribute); a.Type != 1 || a.Value != 1 || a.End != 2 {
		t.Fatalf("underline %+v", a)
	}
	table := dbus.MakeVariant(LookupTable{Name: "IBusLookupTable", Attachments: map[string]dbus.Variant{}, Candidates: []dbus.Variant{}, Labels: []dbus.Variant{}})
	if got := table.Signature().String(); got != "(sa{sv}uubbiavav)" {
		t.Fatalf("IBusLookupTable %s", got)
	}
}

func TestEnginesXML(t *testing.T) {
	var b bytes.Buffer
	EnginesXML(&b, []engine.Engine{hangul.New()})
	if !strings.Contains(b.String(), "<name>sinty-hangul</name>") || !strings.Contains(b.String(), "<language>ko</language>") {
		t.Fatal(b.String())
	}
	for in, want := range map[string]string{"zh-Hans": "zh_CN", "zh-Hant": "zh_TW", "yue": "zh_HK", "ja": "ja"} {
		if Language(in) != want {
			t.Errorf("%s: %s", in, Language(in))
		}
	}
}

func TestAddressFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bus")
	os.WriteFile(file, []byte("# comment\nIBUS_ADDRESS=unix:path=/tmp/x,guid=1\nIBUS_DAEMON_PID=4\n"), 0o644)
	t.Setenv("IBUS_ADDRESS", "")
	t.Setenv("IBUS_ADDRESS_FILE", file)
	if a, err := Address(); err != nil || a != "unix:path=/tmp/x,guid=1" {
		t.Fatalf("%q %v", a, err)
	}
	t.Setenv("IBUS_ADDRESS", "unix:path=/env")
	if a, _ := Address(); a != "unix:path=/env" {
		t.Fatal(a)
	}
}
