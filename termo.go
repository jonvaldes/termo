package termo

import (
	"errors"
	"fmt"
	"syscall"
	"unicode/utf8"

	"code.google.com/p/go.crypto/ssh/terminal"
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
	return nil
}

// Stop restores the terminal to its original state
func Stop() {
	terminal.Restore(syscall.Stdin, oldTermState)
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

// Framebuffer contains the runes to draw
// in the terminal
type Framebuffer struct {
	w, h  int
	chars []rune
}

// NewFramebuffer creates a Framebuffer with the specified size
// and initializes it filling it with blank spaces
func NewFramebuffer(w, h int) *Framebuffer {
	result := &Framebuffer{w, h, make([]rune, w*h)}
	result.Clear()
	return result
}

// Get returns the rune stored in the [x,y] position.
// If coords are outside the framebuffer size, it returns ' '
func (f *Framebuffer) Get(x, y int) rune {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return ' '
	}
	return f.chars[x+y*f.w]
}

// Put sets a rune in the specified position
func (f *Framebuffer) Put(x, y int, r rune) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return
	}
	f.chars[x+y*f.w] = r
}

// PutRect fills a rectangular region with a rune
func (f *Framebuffer) PutRect(x0, y0, w, h int, r rune) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			f.Put(x, y, r)
		}
	}
}

// PutText draws a string from left to right, starting at x0,y0
// There is no wrapping mechanism, and parts of the text outside
// the framebuffer will be ignored.
func (f *Framebuffer) PutText(x0, y0 int, t string) {
	i := 0
	for _, runeValue := range t {
		f.Put(x0+i, y0, runeValue)
		i++
	}
}

// Clear fills the framebuffer with blank spaces
func (f *Framebuffer) Clear() {
	f.PutRect(0, 0, f.w, f.h, ' ')
}

// Flush pushes the current state of the framebuffer to the terminal
func (f *Framebuffer) Flush() {
	fmt.Printf("\033[0;0H")
	for y := 0; y < f.h; y++ {
		if y != 0 {
			fmt.Print("\n")
		}
		s := f.chars[y*f.w : (y+1)*f.w]
		fmt.Print(string(s))
	}
}
