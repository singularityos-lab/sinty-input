package importer

import (
	"strings"
	"testing"
)

func TestRime(t *testing.T) {
	src := "# comment\n---\nname: wubi86\nversion: \"0.7\"\ncolumns:\n  - text\n  - code\n  - weight\n  - stem\nencoder:\n  exclude_patterns:\n    - '^z.*$'\n  rules:\n    - length_equal: 2\n      formula: \"AaAbBaBb\"\n    - length_in_range: [4, 10]\n      formula: \"AaBaCaZa\"\n...\n\n工\ta\t99454797\taa\n#子\tb\t1\n了\tb\t1477224452\tbn\n粤语\tjyut jyu\t5%\n"
	d, err := Rime(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if d.Words("a")[0] != "工" || d.Words("b")[0] != "了" || len(d.Words("b")) != 1 {
		t.Fatalf("entries: %v %v", d.Words("a"), d.Words("b"))
	}
	if d.Score("jyut jyu") != 500 {
		t.Fatalf("percent weight: %d", d.Score("jyut jyu"))
	}
	if d.Get("phrase") != "2:AaAbBaBb 4+:AaBaCaZa" || d.Get("rime") != "wubi86" {
		t.Fatalf("meta: %v", d.Meta)
	}
}

func TestFcitx(t *testing.T) {
	src := ";fcitx Version 0x03 Table file\nKeyCode=abcdefghijklmnopqrstuvwxy\nLength=4\nPinyin=@\nPrompt=&\nConstructPhrase=^\n[Rule]\ne2=p11+p12+p21+p22\na4=p11+p21+p31+n11\n[Data]\n&a 工\n&b 子\na 工\na 戈\naa 式\n@gong 工\n^ad 于\n"
	d, err := Fcitx(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Words("a"); len(got) != 2 || got[0] != "工" || got[1] != "戈" {
		t.Fatalf("order kept: %v", got)
	}
	if d.Lookup("gong") != nil || d.Lookup("@gong") != nil || d.Lookup("^ad") != nil || d.Lookup("ad") != nil {
		t.Fatal("pinyin and construct lines are not entries")
	}
	if d.Get("keys") != "abcdefghijklmnopqrstuvwxy" || d.Get("length") != "4" || d.Get("display") != "a=工 b=子" {
		t.Fatalf("meta: %v", d.Meta)
	}
	if d.Get("phrase") != "2:AaAbBaBb 4+:AaBaCaZa" {
		t.Fatalf("rules: %q", d.Get("phrase"))
	}
	old := "键码=abcdefghijklmnopqrstuvwxyz\n提示=&\n码长=6\n[数据]\n&a 日\na 日\n"
	d, err = Fcitx(strings.NewReader(old))
	if err != nil || d.Words("a")[0] != "日" || d.Get("length") != "6" {
		t.Fatalf("chinese keys: %v %v", err, d)
	}
}

func TestDetect(t *testing.T) {
	cases := map[string]string{
		"@format sinty-dict 1\n":     "sinty",
		"---\nname: x\n...\n":        "rime",
		"KeyCode=abc\n[Data]\na 工\n": "fcitx",
		"键码=abc\n[数据]\n":             "fcitx",
		"just text":                  "",
	}
	for in, want := range cases {
		if got := Detect([]byte(in)); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
	if Compatible("LicenseRef-proprietary") || !Compatible("LGPL-3.0-only") {
		t.Fatal("license check")
	}
}
