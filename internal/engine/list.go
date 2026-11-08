package engine

import "strconv"

const PageSize = 9

type List struct {
	Items    []string
	Selected int
	Aux      string
}

func (l *List) Empty() bool {
	return len(l.Items) == 0
}

func (l *List) Current() string {
	if l.Empty() {
		return ""
	}
	return l.Items[l.Selected]
}

func (l *List) Move(delta int) {
	if l.Empty() {
		return
	}
	n := len(l.Items)
	l.Selected = ((l.Selected+delta)%n + n) % n
}

func (l *List) Turn(next bool) {
	start := l.Selected / PageSize * PageSize
	switch {
	case next && start+PageSize < len(l.Items):
		l.Selected = start + PageSize
	case !next && start > 0:
		l.Selected = start - PageSize
	}
}

func (l *List) Pick(index int) bool {
	i := l.Selected/PageSize*PageSize + index
	if index < 0 || index >= PageSize || i >= len(l.Items) {
		return false
	}
	l.Selected = i
	return true
}

func (l *List) Candidates() *Candidates {
	if l.Empty() {
		return nil
	}
	start := l.Selected / PageSize * PageSize
	end := min(start+PageSize, len(l.Items))
	c := &Candidates{
		Items:    l.Items[start:end],
		Selected: l.Selected - start,
		Previous: start > 0,
		Next:     end < len(l.Items),
		Aux:      l.Aux,
	}
	for i := range c.Items {
		c.Labels = append(c.Labels, strconv.Itoa(i+1))
	}
	return c
}

func (l *List) Clear() {
	l.Items, l.Selected, l.Aux = nil, 0, ""
}
