package termo

import (
	"errors"
	"fmt"
	"syscall"
	"unicode/utf8"

	"code.google.com/p/go.crypto/ssh/terminal"
)

var NotATerminal error = errors.New("not running in a terminal")

var oldTermState *terminal.State

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

func Stop() {
	terminal.Restore(syscall.Stdin, oldTermState)
}

func Size() (int, int, error) {
	return terminal.GetSize(syscall.Stdin)
}

type ScanCode []byte

func (s ScanCode) IsEscapeCode() bool {
	return s[0] == 27 && s[1] == 91
}

func (s ScanCode) EscapeCode() byte {
	return s[2]
}

func (s ScanCode) Rune() rune {
	r, _ := utf8.DecodeRune(s)
	return r
}

func ReadScanCode() (ScanCode, error) {
	s := ScanCode{0, 0, 0, 0, 0, 0}
	_, err := syscall.Read(syscall.Stdin, s)
	return s, err
}

type Framebuffer struct {
	w, h  int
	chars []rune
}

func NewFramebuffer(w, h int) *Framebuffer {
	return &Framebuffer{w, h, make([]rune, w*h)}
}

func (f *Framebuffer) Get(x, y int) rune {
	return f.chars[x+y*f.w]
}

func (f *Framebuffer) Set(x, y int, r rune) {
	f.chars[x+y*f.w] = r
}

func (f *Framebuffer) Draw() {
	fmt.Printf("\033[0;0H")
	for y := 0; y < f.h; y++ {
		s := f.chars[y*f.w : (y+1)*f.w]
		fmt.Println(string(s))
	}
}

func (f *Framebuffer) Rect(x0, y0, w, h int, r rune) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			f.Set(x, y, r)
		}
	}
}

func (f *Framebuffer) Clear() {
	f.Rect(0, 0, f.w, f.h, ' ')
}
