package main

import (
	"github.com/gdamore/tcell"
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
	style    tcell.Style
	screen   tcell.Screen
	modified bool
}

func (t *term) init() (err error) {
	t.style = t.style.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	if t.screen, err = tcell.NewScreen(); err != nil {
		return
	}
	t.screen.SetStyle(t.style)
	if err = t.screen.Init(); err != nil {
		return
	}
	if err = t.reset(); err != nil {
		return
	}
	return
}

func (t *term) reset() (err error) {
	t.screen.Clear()
	t.w, t.h = t.screen.Size()
	t.setCursor(pos{0, 0})
	return
}

func (t *term) flush() {
	if t.modified {
		t.screen.Sync()
		t.w, t.h = t.screen.Size()
		t.setCursor(t.pos)
		t.modified = false
	}
}

func (t *term) setCursor(p pos) {
	t.pos = p
}

func (t *term) showCursor() {
	t.screen.ShowCursor(t.x, t.y)
}

func (t *term) hideCursor() {
	t.screen.HideCursor()
}

func (t *term) writeRune(c rune) {
	if c == '\n' {
		t.y++
	} else if c == '\r' {
		t.x = 0
	} else {
		t.screen.SetContent(t.x, t.y, c, nil, t.style)
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
	t.screen.Fini()
}
