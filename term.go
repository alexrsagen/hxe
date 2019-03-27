package main

import (
	"github.com/nsf/termbox-go"
)

// this file contains terminal setup and user input/output handling

type pos struct {
	x, y int
}

type size struct {
	w, h int
}

type obj struct {
	pos
	size
}

type term struct {
	obj
	modified bool
	fg       termbox.Attribute
	bg       termbox.Attribute
}

func (t *term) init() {
	must(termbox.Init())
	t.reset()
}

func (t *term) reset() {
	t.fg = termbox.ColorWhite
	t.bg = termbox.ColorBlack
	must(termbox.Clear(t.fg, t.bg))
	t.setCursor(pos{0, 0})
	must(termbox.Flush())
	t.w, t.h = termbox.Size()
}

func (t *term) flush() {
	if t.modified {
		must(termbox.Flush())
		t.modified = false
	}
}

func (t *term) setCursor(p pos) {
	t.x, t.y = p.x, p.y
	if t.x < t.w && t.y < t.h {
		termbox.SetCursor(t.x, t.y)
	}
}

func (t *term) writeRune(c rune) {
	if c == '\n' {
		t.y++
	} else if c == '\r' {
		t.x = 0
	} else {
		if t.x < t.w && t.y < t.h {
			termbox.SetCell(t.x, t.y, c, t.fg, t.bg)
		}
		t.x++
	}
	t.setCursor(t.pos)
}

func (t *term) writeWrap(s string) {
	if len(s) == 0 {
		return
	}
	t.modified = true
	for _, c := range s {
		if t.x >= t.w {
			t.x = 0
			t.y++
		}
		if t.y >= t.h {
			t.x, t.y = 0, 0
		}
		t.writeRune(c)
	}
}

func (t *term) writeOverflow(s string) {
	if len(s) == 0 {
		return
	}
	t.modified = true
	for _, c := range s {
		t.writeRune(c)
	}
}

func (t *term) close() {
	termbox.Close()
}
