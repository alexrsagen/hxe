package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gdamore/tcell"
)

type editorArea struct {
	file         *os.File    // current file being edited
	fileStat     os.FileInfo // stat of file
	offset       int64       // byte offset of file to display
	cursorOffset int         // relative to offset
	buffer       []byte      // bytes currently in view, loaded from file at offset
}

// general editor methods

func (a *editorArea) init() (err error) {
	// open file
	if a.file, err = os.OpenFile(app.flags.Filename, os.O_RDWR|os.O_CREATE, 0644); err != nil {
		return
	}
	if a.fileStat, err = a.file.Stat(); err != nil {
		return
	}

	// initialize buffer
	a.buffer = make([]byte, a.bufferSize())
	if err = a.load(); err != nil {
		return
	}

	return
}

func (a *editorArea) onEvent(ev tcell.Event) error {
	switch v := ev.(type) {
	case *tcell.EventResize:
		// resize buffer
		if a.bufferSize() > int64(cap(a.buffer)) {
			a.buffer = make([]byte, a.bufferSize())
		}

		// redraw content
		app.term.hideCursor()
		a.drawStatic()
		a.drawDynamic()
		app.term.setCursor(a.bufferOffsetPos(a.cursorOffset))
		app.term.showCursor()

	case *tcell.EventKey:
		var cursorChanged, pageChanged bool
		switch v.Key() {
		case tcell.KeyLeft:
			// move one byte back
			a.cursorOffset--
			cursorChanged = true
		case tcell.KeyRight:
			// move one byte forward
			a.cursorOffset++
			cursorChanged = true
		case tcell.KeyUp:
			// move one row back
			a.cursorOffset -= app.flags.BytesPerRow
			cursorChanged = true
		case tcell.KeyDown:
			// move one row forward
			a.cursorOffset += app.flags.BytesPerRow
			cursorChanged = true
		case tcell.KeyPgUp:
			// move one page back
			a.cursorOffset -= cap(a.buffer)
			cursorChanged = true
		case tcell.KeyPgDn:
			// move one page forward
			a.cursorOffset += cap(a.buffer)
			cursorChanged = true
		}

		// handle cursor overflow/underflow
		if cursorChanged {
			// correct for cursor underflow
			for a.cursorOffset < 0 {
				// if first page
				if a.offset < int64(cap(a.buffer)) {
					// set offset to zero
					a.cursorOffset = 0
					break
				}

				// go to previous page
				a.offset -= int64(cap(a.buffer))
				pageChanged = true

				// add one page to offset
				a.cursorOffset += cap(a.buffer)
			}

			// correct for cursor overflow
			for a.cursorOffset >= len(a.buffer) {
				// if last page
				if a.fileStat.Size()-a.offset <= int64(len(a.buffer)) {
					// set offset to end of page
					a.cursorOffset = len(a.buffer)
					break
				}

				// do nothing if between len and cap on last page
				if a.cursorOffset < cap(a.buffer) {
					break
				}

				// go to next page
				a.offset += int64(cap(a.buffer))
				pageChanged = true

				// remove one page from offset
				a.cursorOffset -= cap(a.buffer)
			}
		}

		// reload and redraw page if changed
		if pageChanged {
			// correct for cursor underflow on first page
			// correct for cursor overflow on last page
			if a.cursorOffset >= len(a.buffer) && a.fileStat.Size()-a.offset <= int64(len(a.buffer)) {
				// set offset to end of page
				a.cursorOffset = len(a.buffer)
			}

			// reload buffer
			if err := a.load(); err != nil {
				return err
			}

			// redraw dynamic content
			a.clearDynamic()
			a.drawDynamic()
		}

		// reposition cursor
		if cursorChanged || pageChanged {
			app.term.setCursor(a.bufferOffsetPos(a.cursorOffset))
			app.term.showCursor()
		}
	}

	return nil
}

func (a *editorArea) onClose() error {
	return a.file.Close()
}

func (a *editorArea) onFocus() error {
	// draw content
	app.term.hideCursor()
	a.drawStatic()
	a.drawDynamic()
	app.term.setCursor(a.bufferOffsetPos(a.cursorOffset))
	app.term.showCursor()
	return nil
}

func (a *editorArea) onUnfocus() error {
	return nil
}

// data methods

// load fills the editor buffer from the file contents
func (a *editorArea) load() error {
	// resize buffer to max capacity
	a.buffer = a.buffer[:cap(a.buffer)]

	// load bytes into buffer from file at offset
	n, err := a.file.ReadAt(a.buffer, a.offset)
	if err != nil && err != io.EOF {
		return err
	}

	// resize buffer to loaded bytes
	a.buffer = a.buffer[:n]

	return nil
}

// encode converts bytes from UTF-8 to the editor encoding defined in flags
func (a *editorArea) encode(in []byte) ([]byte, error) {
	cm, err := getCharmap(app.flags.Encoding)
	if err != nil {
		return nil, err
	}
	if cm == nil {
		return in, nil
	}
	out, err := cm.NewEncoder().Bytes(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// decode converts bytes from the editor encoding defined in flags to UTF-8
func (a *editorArea) decode(in []byte) ([]byte, error) {
	cm, err := getCharmap(app.flags.Encoding)
	if err != nil {
		return nil, err
	}
	if cm == nil {
		return in, nil
	}
	out, err := cm.NewDecoder().Bytes(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a *editorArea) printableBytesPerRow() int64 {
	const offset = len("Offset(x)")
	base := app.term.w - offset
	charsPerGroup := app.flags.Group*2 + 1
	maxBytesPerRow := (base / charsPerGroup) * app.flags.Group
	return int64(min(app.flags.BytesPerRow, maxBytesPerRow))
}

func (a *editorArea) printableBytes() int64 {
	reservedRows := 2 // header + offset header
	if app.flags.Columns["keys"] {
		reservedRows++ // key reference
	}
	return a.printableBytesPerRow() * int64(app.term.h-reservedRows)
}

func (a *editorArea) bufferSize() int64 {
	return min64(a.printableBytes(), a.fileStat.Size())
}

func (a *editorArea) bufferOffsetPos(bufferOffset int) pos {
	row := bufferOffset / app.flags.BytesPerRow
	if bufferOffset == len(a.buffer) && bufferOffset%app.flags.BytesPerRow == 0 {
		return pos{9 + app.flags.BytesPerRow*2 + app.flags.BytesPerRow/app.flags.Group, 1 + row}
	}
	rowByte := bufferOffset - row*app.flags.BytesPerRow
	col := rowByte*2 + rowByte/app.flags.Group
	return pos{10 + col, 2 + row}
}

// drawing methods

// drawStatic draws static content to the terminal
func (a *editorArea) drawStatic() {
	var i, j, pad int

	// invert foreground and background
	app.term.style = app.term.style.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite)
	app.term.screen.SetStyle(app.term.style)

	// reset cursor position
	app.term.setCursor(pos{0, 0})

	// draw editor info
	app.term.writeOverflow("  hxe ")
	app.term.writeOverflow(version)

	// draw file info
	headerPadding := (app.term.w - app.term.x - 2) / 2
	app.term.writeOverflow(strings.Repeat(" ", headerPadding))
	app.term.writeOverflow(a.fileStat.Name())
	app.term.writeOverflow(strings.Repeat(" ", headerPadding))

	// draw background for rest of row
	for app.term.x < app.term.w {
		app.term.writeOverflow(" ")
	}

	// set new foreground and background
	app.term.style = app.term.style.Foreground(tcell.ColorBlack).Background(tcell.ColorBlue)
	app.term.screen.SetStyle(app.term.style)

	// draw key reference
	if app.flags.Columns["keys"] {
		// set cursor position to last row
		app.term.setCursor(pos{0, app.term.h - 1})

		// draw keys
		a.drawKey("F10", "Quit")

		// draw background for rest of row
		for app.term.x < app.term.w {
			app.term.writeOverflow(" ")
		}
	}

	// draw hex data view
	if app.flags.Columns["hex"] {
		// reset cursor position
		app.term.setCursor(pos{0, 0})

		// draw offset base
		switch app.flags.OffsetBase {
		case "hex":
			pad = 1 + (app.flags.BytesPerRow-(app.flags.BytesPerRow%0xFF))/0xFF
			app.term.writeOverflow(strings.Repeat("\n", pad) + "Offset(h) ")
		case "dec":
			pad = 1 + (app.flags.BytesPerRow-(app.flags.BytesPerRow%99))/99
			app.term.writeOverflow(strings.Repeat("\n", pad) + "Offset(d) ")
		case "oct":
			pad = 1 + (app.flags.BytesPerRow-(app.flags.BytesPerRow%077))/077
			app.term.writeOverflow(strings.Repeat("\n", pad) + "Offset(o) ")
		}

		// draw offsets
		for i = 1; j < app.flags.BytesPerRow; i++ {
			if app.term.x >= app.term.w {
				break
			}
			if i%app.flags.Group == 0 {
				curx, curj := app.term.x, j
				switch app.flags.OffsetBase {
				case "hex":
					curpad := (curj - (curj % 0xFF)) / 0xFF
					for ; curpad > 0; curpad-- {
						app.term.setCursor(pos{curx, pad - curpad})
						app.term.writeOverflow("FF")
						app.term.writeOverflow(strings.Repeat(" ", app.flags.Group*2-1))
						curj -= 0xFF
					}
					app.term.setCursor(pos{curx, pad})
					app.term.writeOverflow(fmt.Sprintf("%02X", curj))

				case "dec":
					curpad := (curj - (curj % 99)) / 99
					for ; curpad > 0; curpad-- {
						app.term.setCursor(pos{curx, pad - curpad})
						app.term.writeOverflow("99")
						app.term.writeOverflow(strings.Repeat(" ", app.flags.Group*2-1))
						curj -= 99
					}
					app.term.setCursor(pos{curx, pad})
					app.term.writeOverflow(fmt.Sprintf("%02d", curj))

				case "oct":
					curpad := (curj - (curj % 077)) / 077
					for ; curpad > 0; curpad-- {
						app.term.setCursor(pos{curx, pad - curpad})
						app.term.writeOverflow("77")
						app.term.writeOverflow(strings.Repeat(" ", app.flags.Group*2-1))
						curj -= 077
					}
					app.term.setCursor(pos{curx, pad})
					app.term.writeOverflow(fmt.Sprintf("%02o", curj))
				}

				j += app.flags.Group
				app.term.writeOverflow(strings.Repeat(" ", app.flags.Group*2-1))
			}
		}

		// draw separator for text column
		if app.flags.Columns["text"] {
			app.term.writeOverflow(" ")
		}
	}

	// draw text data view
	if app.flags.Columns["text"] {
		app.term.writeOverflow("Decoded text")
	}

	// draw background for rest of row
	for app.term.x < app.term.w {
		app.term.writeOverflow(" ")
	}

	// restore foreground and background
	app.term.style = app.term.style.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	app.term.screen.SetStyle(app.term.style)

	// move cursor to start of new line
	app.term.writeOverflow("\r\n")
}

func (a *editorArea) drawDynamic() {
	// reset cursor position
	app.term.setCursor(pos{0, 2})

	for offset := 0; offset < len(a.buffer); offset += app.flags.BytesPerRow {
		// draw hex data view
		if app.flags.Columns["hex"] {
			a.drawOffset(a.offset + int64(offset))
			a.drawBytes(a.buffer[offset:min(offset+app.flags.BytesPerRow, len(a.buffer))])

			// draw separator for text column
			if app.flags.Columns["text"] {
				app.term.writeOverflow(" ")
			}
		}

		// draw text data view
		if app.flags.Columns["text"] {
			a.drawText(a.buffer[offset:min(offset+app.flags.BytesPerRow, len(a.buffer))])
		}

		// move cursor to start of new line
		app.term.writeOverflow("\r\n")
	}
}

func (a *editorArea) clearDynamic() {
	// empty dynamic area
	for i := 2; i < app.term.h-1; i++ {
		// set cursor position
		app.term.setCursor(pos{0, i})

		// draw empty row
		app.term.writeOverflow(strings.Repeat(" ", app.term.w))
	}
}

func (a *editorArea) drawKey(key, desc string) {
	// invert foreground and background
	app.term.style = app.term.style.Foreground(tcell.ColorBlue).Background(tcell.ColorBlack)
	app.term.screen.SetStyle(app.term.style)

	// draw key
	app.term.writeOverflow(key)

	// restore foreground and background
	app.term.style = app.term.style.Foreground(tcell.ColorBlack).Background(tcell.ColorBlue)
	app.term.screen.SetStyle(app.term.style)

	// draw description and padding for next item
	app.term.writeOverflow(desc + " ")
}

func (a *editorArea) drawOffset(offset int64) {
	if app.flags.Columns["hex"] {
		switch app.flags.OffsetBase {
		case "hex":
			app.term.writeOverflow(fmt.Sprintf("%08X  ", offset))
		case "dec":
			app.term.writeOverflow(fmt.Sprintf("%08d  ", offset))
		case "oct":
			app.term.writeOverflow(fmt.Sprintf("%08o  ", offset))
		}
	}
}

func (a *editorArea) drawBytes(b []byte) {
	// draw bytes in current row
	for i := 0; i < len(b); i += app.flags.Group {
		app.term.writeOverflow(strings.ToUpper(hex.EncodeToString(b[i:min(i+app.flags.Group, len(b))])) + " ")
	}

	// draw background for rest of row
	for i := 0; i < app.flags.BytesPerRow-len(b); i++ {
		app.term.writeOverflow(strings.Repeat(" ", app.flags.Group*2+1))
	}
}

func (a *editorArea) drawText(b []byte) {
	encoded := app.must(a.encode(b)).([]byte)
	for _, c := range encoded {
		if c > 31 && c < 127 {
			app.term.writeOverflow(string(c))
		} else {
			app.term.writeOverflow(".")
		}
	}
}
