package termo

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/jonvaldes/termo/terminal"
)

// ErrNotATerminal is the error returned when running
// termo in an unsupported environment
var ErrNotATerminal = errors.New("not running in a terminal")

var oldTermState *terminal.State

// Control sequences documentation: http://www.xfree86.org/current/ctlseqs.html

// Init initializes termo to work with the terminal
func Init() error {
	if !terminal.IsTerminal(syscall.Stdin) {
		return ErrNotATerminal
	}
	var err error
	oldTermState, err = terminal.MakeRaw(syscall.Stdin)
	if err != nil {
		panic(err)
	}
	HideCursor()
	return nil
}

// Stop restores the terminal to its original state
func Stop() {
	terminal.Restore(syscall.Stdin, oldTermState)
	ShowCursor()
	fmt.Printf("\033[?1003l") // Reset mouse
}

// HideCursor makes the cursor invisible
func HideCursor() {
	fmt.Printf("\033[?25l")
}

// ShowCursor makes the cursor visible
func ShowCursor() {
	fmt.Printf("\033[?25h")
}

var cursorPos [2]int

// SetCursor positions the cursor at the specified coordinates.
// Cursor visibility is not affected.
func SetCursor(x, y int) {
	cursorPos[0] = x
	cursorPos[1] = y
	fmt.Printf("\033[%d;%dH", y+1, x+1)
}

// EnableMouseEvents makes mouse events start
// arriving through the input read loop
func EnableMouseEvents() {
	fmt.Printf("\033[?1003h")
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
	return len(s) > 2 && s[0] == 27 && s[1] == 91
}

// EscapeCode returns the escape code for a keypress
func (s ScanCode) EscapeCode() byte {
	return s[2]
}

// IsMouseMoveEvent returns wether it is a mouse move event
func (s ScanCode) IsMouseMoveEvent() bool {
	return len(s) == 6 && s.IsEscapeCode() && s[2] == 77 && s[3] == 67
}

// IsMouseDownEvent returns wether it is a mouse button down event
func (s ScanCode) IsMouseDownEvent() bool {
	return len(s) == 6 && s.IsEscapeCode() && s[2] == 77 && s[3] == 32
}

// IsMouseUpEvent returns wether it is a mouse button up event
func (s ScanCode) IsMouseUpEvent() bool {
	return len(s) == 6 && s.IsEscapeCode() && s[2] == 77 && s[3] == 35
}

// MouseCoords returns data for the mouse position.
// Returned coords start at [0,0] for upper-left corner
func (s ScanCode) MouseCoords() (int, int) {
	return int(s[4] - 33), int(s[5] - 33)
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

// StartKeyReadLoop runs a goroutine that
// keeps reading terminal input forever.
// It returns events through the keyChan param, and
// errors through the errChan parameter
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

// Attribute holds data for each
// possible visualization mode
type Attribute int

// Attributes for different character
// visualization modes
const (
	AttrNone  Attribute = 0
	AttrBold  Attribute = 1
	AttrDim   Attribute = 2
	AttrUnder Attribute = 4
	AttrBlink Attribute = 5
	AttrRev   Attribute = 7
	AttrHid   Attribute = 8
)

// Color holds character color information
type Color int

// Different colors to use as attributes
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

// Light returns the "ligther" version for that color
func (c Color) Light() Color {
	return c + 60
}

func background(c Color) Color {
	return c + 10
}

// CellState holds all the attributes for a cell
type CellState struct {
	Attrib  Attribute
	FGColor Color
	BGColor Color
}

// Predefined attributes
var (
	StateDefault     = CellState{Attrib: AttrNone, FGColor: ColorDefault, BGColor: ColorDefault}
	BoldWhiteOnBlack = CellState{Attrib: AttrBold, FGColor: ColorGray.Light(), BGColor: ColorBlack}
	BoldBlackOnWhite = CellState{Attrib: AttrBold, FGColor: ColorBlack, BGColor: ColorGray.Light()}
)

type cell struct {
	state CellState
	r     rune
}

// Framebuffer contains the runes and attributes
// that will be drawn in the terminal
type Framebuffer struct {
	w, h  int
	chars []cell
}

// NewFramebuffer creates a Framebuffer with the specified size
// and initializes it filling it with blank spaces and default
// attributes
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

// Set sets a rune in the specified position with the specified attributes
func (f *Framebuffer) Set(x, y int, s CellState, r rune) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return
	}
	f.chars[x+y*f.w].r = r
	f.chars[x+y*f.w].state = s
}

// SetRune sets a rune in the specified position without modifying its attributes
func (f *Framebuffer) SetRune(x, y int, r rune) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return
	}
	f.chars[x+y*f.w].r = r
}

// SetRect fills a rectangular region with a rune and state
func (f *Framebuffer) SetRect(x0, y0, w, h int, s CellState, r rune) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			f.Set(x, y, s, r)
		}
	}
}

// AttribRect sets the attributes for a rectangular region
// without changing the runes
func (f *Framebuffer) AttribRect(x0, y0, w, h int, s CellState) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			if x >= 0 && y >= 0 && x < f.w && y < f.h {
				f.chars[x+y*f.w].state = s
			}
		}
	}
}

var singleWidthCharset = []rune{'─', '│', '┌', '┐', '└', '┘'}
var doubleWidthCharset = []rune{'═', '║', '╔', '╗', '╚', '╝'}

// ASCIIRect draws an ASCII rectangle. It can either be
// single-width (─) or double-width (═). It can also clear
// the inner part of the rectangle, if desired.
func (f *Framebuffer) ASCIIRect(x0, y0, w, h int, doubleWidth bool, clearInside bool) {
	c := singleWidthCharset
	if doubleWidth {
		c = doubleWidthCharset
	}

	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			var r rune
			if x == x0 {
				if y == y0 {
					r = c[2]
				} else if y == y0+h-1 {
					r = c[4]
				} else {
					r = c[1]
				}
			} else if x == x0+w-1 {
				if y == y0 {
					r = c[3]
				} else if y == y0+h-1 {
					r = c[5]
				} else {
					r = c[1]
				}
			} else if y == y0 || y == y0+h-1 {
				r = c[0]
			} else {
				if !clearInside {
					continue
				} else {
					r = ' '
				}
			}

			f.SetRune(x, y, r)
		}
	}
}

// SetText draws a string from left to right, and top-to bottom,
// starting at x0,y0.
// There is no wrapping mechanism, and parts of the text outside
// the framebuffer will be ignored. Attributes for written cells
// will remain unchanged.
func (f *Framebuffer) SetText(x0, y0 int, t string) {
	i := 0
	for _, runeValue := range t {
		if runeValue == '\n' {
			i = 0
			y0++
			continue
		}
		f.SetRune(x0+i, y0, runeValue)
		i++
	}
}

// CenterText draws a string from left to right and top-to-bottom,
// starting at x-len(t)/2,y0.
// There is no wrapping mechanism, and parts of the text outside
// the framebuffer will be ignored. Attributes for written cells
// will remain unchanged.
func (f *Framebuffer) CenterText(x, y0 int, t string) {
	lines := strings.Split(t, "\n")
	for y, s := range lines {
		i := 0
		for _, runeValue := range s {
			f.SetRune(x+i-len(s)/2, y0+y, runeValue)
			i++
		}
	}
}

// AttribText draws a string from left to right, starting at x0,y0
// There is no wrapping mechanism, and parts of the text outside
// the framebuffer will be ignored. This call will also change the
// written cells' attributes to the specified ones.
func (f *Framebuffer) AttribText(x0, y0 int, s CellState, t string) {
	i := 0
	for _, runeValue := range t {
		if runeValue == '\n' {
			i = 0
			y0++
			continue
		}
		f.Set(x0+i, y0, s, runeValue)
		i++
	}
}

// Clear fills the framebuffer with blank spaces and default attributes
func (f *Framebuffer) Clear() {
	f.SetRect(0, 0, f.w, f.h, StateDefault, ' ')
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

	// Move cursor to correct position
	fmt.Printf("\033[%d;%dH", cursorPos[1]+1, cursorPos[0]+1)
}
