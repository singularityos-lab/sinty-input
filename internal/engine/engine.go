package engine

const (
	KeySpace     = 0x20
	KeyBackSpace = 0xff08
	KeyTab       = 0xff09
	KeyReturn    = 0xff0d
	KeyEscape    = 0xff1b
	KeyLeft      = 0xff51
	KeyUp        = 0xff52
	KeyRight     = 0xff53
	KeyDown      = 0xff54
	KeyPageUp    = 0xff55
	KeyPageDown  = 0xff56
	KeyKPEnter   = 0xff8d

	StateShift   = 1 << 0
	StateLock    = 1 << 1
	StateControl = 1 << 2
	StateAlt     = 1 << 3
	StateSuper   = 1 << 6
)

type Key struct {
	Sym     uint32
	Code    uint32
	State   uint32
	Release bool
}

func (k Key) Shortcut() bool {
	return k.State&(StateControl|StateAlt|StateSuper) != 0
}

func (k Key) Printable() (rune, bool) {
	if k.Sym >= 0x20 && k.Sym <= 0x7e {
		return rune(k.Sym), true
	}
	return 0, false
}

func (k Key) Digit() (int, bool) {
	if k.Sym >= '1' && k.Sym <= '9' {
		return int(k.Sym - '1'), true
	}
	return 0, false
}

type Candidates struct {
	Items    []string
	Labels   []string
	Selected int
	Previous bool
	Next     bool
	Aux      string
}

type Output interface {
	Commit(text string)
	Preedit(text string, cursor int)
	Candidates(c *Candidates)
}

type Info struct {
	Name     string
	Label    string
	Symbol   string
	Language string
}

type Engine interface {
	Info() Info
	Key(k Key, out Output) bool
	Select(index int, out Output)
	Page(next bool, out Output)
	Reset(out Output)
	Flush(out Output)
}

func Runes(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

type Options struct {
	Conversion string
	FullWidth  bool
	Learn      bool
	Dictionary string
}

func DefaultOptions() Options {
	return Options{FullWidth: true, Learn: true}
}

type Configurable interface {
	Configure(o Options)
	Options() Options
}
