termo
=====

Simple ncurses-style terminal drawing lib in Go.

It tries to be simple to use while being more reliable than go-termbox.

While go-termbox writes each character individually to the terminal, termo
keeps an internal "framebuffer", and then flushes the whole
framebuffer to the terminal. This is less performant, but more reliable.

API
---

The steps to use termo in your program are the following:
- Initialize termo
```go    
    termo.Init()
```
- Defer the termo shutdown to restore the terminal state:
```go
    defer termo.Stop()
```
- Get the terminal size:
```go
    w, h, _ := termo.Size()
```
- Create a framebuffer:
```go
    fb := termo.NewFramebuffer(w,h)
```
- Draw something to the framebuffer:
```go
    fb.ASCIIRect(3, 3, 20, 20, false, false) // Draw a 20x20 ASCII rectangle
    fb.SetText(4, 4, "I'm now using termo!") // Draw text
    fb.AttribRect(0, 4, w, 1, termo.BoldWhiteOnBlack) // Set character colors/attributes
```
- Flush the framebuffer to the terminal:
```go
    fb.Flush()
```
And that's it!

For more advanced usage, you can check out an example program here: 
https://github.com/jonvaldes/termo_example

Also, here's the full package documentation: https://godoc.org/github.com/jonvaldes/termo

_Note_: This project has only been tested in OSX, but should work in any unix, 
VT100-style terminal. Some advanced features like mouse support might only work
in some terminals (for example, the default OSX terminal doesn't support mouse
events, while iTerm2 does).

