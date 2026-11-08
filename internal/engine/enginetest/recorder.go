package enginetest

import (
	"strings"

	"github.com/singularityos-lab/sinty-input/internal/engine"
)

type Recorder struct {
	Committed strings.Builder
	Text      string
	Cursor    int
	List      *engine.Candidates
}

func (r *Recorder) Commit(text string) {
	r.Committed.WriteString(text)
}

func (r *Recorder) Preedit(text string, cursor int) {
	r.Text, r.Cursor = text, cursor
}

func (r *Recorder) Candidates(c *engine.Candidates) {
	r.List = c
}

func Type(e engine.Engine, r *Recorder, text string) []bool {
	var handled []bool
	for _, c := range text {
		handled = append(handled, e.Key(engine.Key{Sym: uint32(c)}, r))
	}
	return handled
}

func Press(e engine.Engine, r *Recorder, sym uint32) bool {
	return e.Key(engine.Key{Sym: sym}, r)
}
