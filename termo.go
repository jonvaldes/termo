package termo

import (
	"errors"
	"fmt"
	"syscall"
	"unicode/utf8"

	"github.com/jonvaldes/termo/terminal"
)

// NotATerminal is the error returned when running
// termo in an unsupported environment
var NotATerminal error = errors.New("not running in a terminal")

var oldTermState *terminal.State

// Init initializes termo to work with the terminal
func Init() error {
	if !terminal.IsTerminal(syscall.Stdin) {
		return NotATerminal
	}
	var err error
	oldTermState, err = terminal.MakeRaw(syscall.Stdin)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\033[?25l")
	return nil
}

// Stop restores the terminal to its original state
func Stop() {
	terminal.Restore(syscall.Stdin, oldTermState)
	fmt.Printf("\033[?25h")
}

// Size returns the current size of the terminal
func Size() (int, int, error) {
	return terminal.GetSize(syscall.Stdin)
}

// ScanCode contains data for a terminal keypress
type ScanCode []byte

// IsEscapeCode returns true if the terminal
// considers it an escape code
func (s ScanCode) IsEscapeCode() bool {
	return s[0] == 27 && s[1] == 91
}

// EscapeCode returns the escape code for a keypress
func (s ScanCode) EscapeCode() byte {
	return s[2]
}

// Rune returns the actual key pressed (only for
// non-escapecode keypresses)
func (s ScanCode) Rune() rune {
	r, _ := utf8.DecodeRune(s)
	return r
}

// ReadScanCode reads a keypress from stdin.
// It will block until it can read something
func ReadScanCode() (ScanCode, error) {
	s := ScanCode{0, 0, 0, 0, 0, 0}
	_, err := syscall.Read(syscall.Stdin, s)
	return s, err
}

func StartKeyReadLoop(keyChan chan<- ScanCode, errChan chan<- error) {
	go func() {
		for {
			s, err := ReadScanCode()
			if err != nil {
				errChan <- err
				return
			}
			keyChan <- s
		}
	}()
}

type Attribute int

const (
	AttrNone  Attribute = 0
	AttrBold  Attribute = 1
	AttrDim   Attribute = 2
	AttrUnder Attribute = 4
	AttrBlink Attribute = 5
	AttrRev   Attribute = 7
	AttrHid   Attribute = 8
)

type Color int

const (
	ColorBlack Color = 30 + iota
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorGray
	ColorDefault Color = 39
)

func (c Color) Light() Color {
	return c + 60
}

func background(c Color) Color {
	return c + 10
}

type CellState struct {
	Attrib  Attribute
	FGColor Color
	BGColor Color
}

type cell struct {
	state CellState
	r     rune
}

// Framebuffer contains the runes to draw
// in the terminal
type Framebuffer struct {
	w, h  int
	chars []cell
}

// NewFramebuffer creates a Framebuffer with the specified size
// and initializes it filling it with blank spaces
func NewFramebuffer(w, h int) *Framebuffer {
	result := &Framebuffer{w, h, make([]cell, w*h)}
	result.Clear()
	return result
}

// Get returns the rune stored in the [x,y] position.
// If coords are outside the framebuffer size, it returns ' '
func (f *Framebuffer) Get(x, y int) (rune, CellState) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return ' ', CellState{AttrNone, ColorDefault, ColorDefault}
	}
	c := f.chars[x+y*f.w]
	return c.r, c.state
}

// Put sets a rune in the specified position
func (f *Framebuffer) Put(x, y int, s CellState, r rune) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return
	}
	f.chars[x+y*f.w].r = r
	f.chars[x+y*f.w].state = s
}

// PutRect fills a rectangular region with a rune
func (f *Framebuffer) PutRect(x0, y0, w, h int, s CellState, r rune) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			f.Put(x, y, s, r)
		}
	}
}

// PutText draws a string from left to right, starting at x0,y0
// There is no wrapping mechanism, and parts of the text outside
// the framebuffer will be ignored.
func (f *Framebuffer) PutText(x0, y0 int, s CellState, t string) {
	i := 0
	for _, runeValue := range t {
		f.Put(x0+i, y0, s, runeValue)
		i++
	}
}

// Clear fills the framebuffer with blank spaces
func (f *Framebuffer) Clear() {
	f.PutRect(0, 0, f.w, f.h, CellState{Attrib: AttrNone, FGColor: ColorDefault, BGColor: ColorDefault}, ' ')
}

// Flush pushes the current state of the framebuffer to the terminal
func (f *Framebuffer) Flush() {
	fmt.Printf("\033[0;0H")
	for y := 0; y < f.h; y++ {
		if y != 0 {
			fmt.Print("\n")
		}
		for x := 0; x < f.w; x++ {
			c := f.chars[y*f.w+x]
			if c.r < 32 {
				continue
			}
			fmt.Printf("\033[%d;%d;%dm%c\033[0m", c.state.Attrib, c.state.FGColor, background(c.state.BGColor), c.r)
		}
	}
	fmt.Printf("\033[0m")
}
