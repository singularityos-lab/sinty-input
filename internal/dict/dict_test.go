package dict

import "testing"

func TestLookup(t *testing.T) {
	cases := []struct {
		table *Table
		key   string
		first string
	}{
		{Japanese, "にほん", "日本"},
		{Japanese, "かk", "書"},
		{Chinese, "shi", "是"},
		{Chinese, "zhong guo", "中国"},
	}
	for _, c := range cases {
		got := c.table.Lookup(c.key)
		if len(got) == 0 || got[0] != c.first {
			t.Errorf("%q: got %v, want %q first", c.key, got, c.first)
		}
	}
	if Japanese.Lookup("ゔゔゔゔ") != nil {
		t.Error("unknown reading must have no candidates")
	}
}
