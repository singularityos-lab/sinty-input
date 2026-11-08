package engine

import (
	"fmt"
	"testing"
)

func numbers(n int) *List {
	l := &List{}
	for i := range n {
		l.Items = append(l.Items, fmt.Sprint(i))
	}
	return l
}

func TestPaging(t *testing.T) {
	l := numbers(20)
	c := l.Candidates()
	if len(c.Items) != 9 || c.Previous || !c.Next || c.Labels[8] != "9" {
		t.Fatalf("first page %+v", c)
	}
	l.Turn(true)
	l.Turn(true)
	c = l.Candidates()
	if len(c.Items) != 2 || !c.Previous || c.Next || l.Current() != "18" {
		t.Fatalf("last page %+v current %s", c, l.Current())
	}
	l.Turn(true)
	if l.Current() != "18" {
		t.Fatal("turning past the last page must stay")
	}
	if !l.Pick(1) || l.Current() != "19" || l.Pick(2) {
		t.Fatal("pick on the last page")
	}
}

func TestMoveWraps(t *testing.T) {
	l := numbers(3)
	l.Move(-1)
	if l.Current() != "2" {
		t.Fatal(l.Current())
	}
	l.Move(1)
	if l.Current() != "0" {
		t.Fatal(l.Current())
	}
}
