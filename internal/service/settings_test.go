package service

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"

	"github.com/singularityos-lab/sinty-input/internal/config"
	"github.com/singularityos-lab/sinty-input/internal/registry"
	"github.com/singularityos-lab/sinty-input/internal/userdict"
)

func bus(t *testing.T) string {
	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("dbus-daemon not installed")
	}
	cmd := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address")
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	addr, _ := bufio.NewReader(out).ReadString('\n')
	return strings.TrimSpace(addr)
}

func TestSettingsOverTheBus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SINTY_INPUT_USER_DIR", home)
	t.Setenv("SINTY_INPUT_CONFIG", filepath.Join(home, "config.json"))
	userdict.Forget()
	defer userdict.Forget()
	addr := bus(t)
	server, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	svc, err := Export(server, registry.Engines())
	if err != nil {
		t.Fatal(err)
	}
	svc.Rebuild = registry.Engines
	server.RequestName(Name, dbus.NameFlagDoNotQueue)
	client, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	obj := client.Object(Name, Path)
	call := func(method string, out []interface{}, args ...interface{}) {
		t.Helper()
		c := obj.Call(Iface+"."+method, 0, args...)
		if c.Err != nil {
			t.Fatalf("%s: %v", method, c.Err)
		}
		if out != nil {
			if err := c.Store(out...); err != nil {
				t.Fatalf("%s: %v", method, err)
			}
		}
	}

	var methods []struct {
		Name, Label, Symbol, Language string
		Options                       map[string]dbus.Variant
	}
	call("Methods", []interface{}{&methods})
	names := map[string]map[string]dbus.Variant{}
	for _, m := range methods {
		names[m.Name] = m.Options
	}
	for _, n := range []string{"pinyin", "shuangpin-xiaohe", "shuangpin-ziranma", "shuangpin-microsoft", "shuangpin-sogou", "shuangpin-abc", "shuangpin-ziguang",
		"zhuyin", "jyutping", "wubi86", "wubi98", "wubi-ng", "cangjie3", "cangjie5", "quick", "array30", "zhengma", "kana", "hangul"} {
		if _, ok := names[n]; !ok {
			t.Errorf("method %s missing", n)
		}
	}
	if names["zhuyin"]["conversion"].Value() != "taiwan" || names["wubi86"]["kind"].Value() != "table" || names["pinyin"]["kind"].Value() != "phonetic" {
		t.Fatalf("defaults: %v %v", names["zhuyin"], names["wubi86"])
	}
	if _, ok := names["hangul"]["conversion"]; ok {
		t.Fatal("conversion is only for Chinese")
	}

	call("Configure", nil, "cangjie5", map[string]dbus.Variant{"conversion": dbus.MakeVariant("simplified"), "fullwidth": dbus.MakeVariant(false)})
	if c := obj.Call(Iface+".Configure", 0, "cangjie5", map[string]dbus.Variant{"conversion": dbus.MakeVariant("klingon")}); c.Err == nil {
		t.Fatal("unknown conversion accepted")
	}
	if c := obj.Call(Iface+".Configure", 0, "cangjie5", map[string]dbus.Variant{"dictionary": dbus.MakeVariant("missing")}); c.Err == nil {
		t.Fatal("missing dictionary accepted")
	}
	saved := config.Load().Methods["cangjie5"]
	if saved.Conversion != "simplified" || saved.Wide() {
		t.Fatalf("saved %+v", saved)
	}
	call("SelectEngine", nil, "cangjie5")
	var handled bool
	for _, k := range "yrcru " {
		call("ProcessKey", []interface{}{&handled}, uint32(k), uint32(0), uint32(0), false)
	}
	call("ProcessKey", []interface{}{&handled}, uint32(','), uint32(0), uint32(0), false)
	if handled {
		t.Fatal("full width punctuation is off, the comma belongs to the application")
	}

	call("UserSet", nil, "pinyin", "sin ti", "辛缇", int32(3))
	var learned []struct {
		Code, Text string
		Weight     int32
	}
	call("UserEntries", []interface{}{&learned}, "pinyin")
	if len(learned) != 1 || learned[0].Text != "辛缇" || learned[0].Weight != 3 {
		t.Fatalf("learned %+v", learned)
	}
	export := filepath.Join(t.TempDir(), "mine.sdict")
	call("UserExport", nil, "pinyin", export)
	call("UserClear", nil, "pinyin")
	call("UserEntries", []interface{}{&learned}, "pinyin")
	if len(learned) != 0 {
		t.Fatalf("cleared %+v", learned)
	}
	var n uint32
	call("UserImport", []interface{}{&n}, "pinyin", export)
	if n != 1 {
		t.Fatalf("imported %d", n)
	}
	call("SelectEngine", nil, "pinyin")
	for _, k := range "sinti " {
		call("ProcessKey", []interface{}{&handled}, uint32(k), uint32(0), uint32(0), false)
	}
	var dicts []struct {
		Name, Type, Language, Title, License string
		User                                 bool
	}
	call("Dictionaries", []interface{}{&dicts})
	if len(dicts) < 16 {
		t.Fatalf("dictionaries %d", len(dicts))
	}

	mine := filepath.Join(home, "dict")
	os.MkdirAll(mine, 0o755)
	os.WriteFile(filepath.Join(mine, "my-table.sdict"), []byte("@format sinty-dict 1\n@name My Table\n@type table\n@language zh-Hans\n@license CC0-1.0\n@keys ab\n@length 2\nab\t好\n"), 0o644)
	t.Setenv("SINTY_INPUT_DICT_PATH", mine+string(os.PathListSeparator)+os.Getenv("SINTY_INPUT_DICT_PATH"))
	call("Reload", nil)
	call("Methods", []interface{}{&methods})
	found := false
	for _, m := range methods {
		if m.Name == "table-my-table" && m.Label == "My Table" {
			found = true
		}
	}
	if !found {
		t.Fatal("a table added by the user becomes a method after Reload")
	}
}
