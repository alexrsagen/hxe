package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nsf/termbox-go"
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

func (a *editorArea) onEvent(ev termbox.Event) error {
	switch ev.Type {
	case termbox.EventResize:
		// resize buffer
		if a.bufferSize() > int64(cap(a.buffer)) {
			a.buffer = make([]byte, a.bufferSize())
		}

		// redraw content
		a.drawStatic()
		a.drawDynamic()
		app.term.setCursor(a.bufferOffsetPos(a.cursorOffset))
	case termbox.EventKey:
		currentCursorOffset := a.cursorOffset
		switch ev.Key {
		case termbox.KeyArrowLeft:
			// move one byte back
			a.cursorOffset--
		case termbox.KeyArrowRight:
			// move one byte forward
			a.cursorOffset++
		case termbox.KeyArrowUp:
			// move one row back
			a.cursorOffset -= app.flags.BytesPerRow
		case termbox.KeyArrowDown:
			// move one row forward
			a.cursorOffset += app.flags.BytesPerRow
		}
		if a.cursorOffset != currentCursorOffset {
			for a.cursorOffset < 0 {
				// if first page, set offset to zero
				if a.offset < int64(len(a.buffer)) {
					a.cursorOffset = 0
					break
				}
				// go to previous page
				// add one page to offset
				a.cursorOffset += len(a.buffer)
			}
			for a.cursorOffset > len(a.buffer) {
				// if last page, set offset to end of page
				if a.fileStat.Size()-a.offset <= int64(len(a.buffer)) {
					a.cursorOffset = len(a.buffer)
					break
				}
				// go to next page
				// remove one page from offset
				a.cursorOffset -= len(a.buffer)
			}
			app.term.setCursor(a.bufferOffsetPos(a.cursorOffset))
		}
	}

	return nil
}

func (a *editorArea) onClose() error {
	return a.file.Close()
}

func (a *editorArea) onFocus() error {
	// draw content
	a.drawStatic()
	a.drawDynamic()
	app.term.setCursor(a.bufferOffsetPos(a.cursorOffset))
	return nil
}

func (a *editorArea) onUnfocus() error {
	return nil
}

// data methods

// load fills the editor buffer from the file contents
func (a *editorArea) load() error {
	if _, err := a.file.ReadAt(a.buffer, a.offset); err != nil && err != io.EOF {
		return err
	}
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
	return min64(a.offset+a.printableBytes(), a.fileStat.Size())
}

func (a *editorArea) bufferOffsetPos(bufferOffset int) pos {
	row := bufferOffset / app.flags.BytesPerRow
	rowByte := bufferOffset - row*app.flags.BytesPerRow
	col := rowByte*2 + rowByte/app.flags.Group
	return pos{10 + col, 2 + row}
}

// drawing methods

// drawStatic draws static content to the terminal
func (a *editorArea) drawStatic() {
	var i, j, pad int

	// invert foreground and background
	app.term.fg, app.term.bg = app.term.bg, app.term.fg

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

	// restore foreground and background
	app.term.fg, app.term.bg = app.term.bg, app.term.fg

	// set new foreground and background
	fg, bg := app.term.fg, app.term.bg
	app.term.fg, app.term.bg = termbox.ColorBlack, termbox.ColorCyan

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
	app.term.fg, app.term.bg = fg, bg

	// move cursor to start of new line
	app.term.writeOverflow("\r\n")
}

func (a *editorArea) drawDynamic() {
	bytesPerRow := int64(app.flags.BytesPerRow)
	offset := a.offset
	size := a.bufferSize()

	for ; offset < size; offset += bytesPerRow {
		// draw hex data view
		if app.flags.Columns["hex"] {
			a.drawOffset(offset)
			a.drawBytes(a.buffer[offset:min64(offset+bytesPerRow, size)])

			// draw separator for text column
			if app.flags.Columns["text"] {
				app.term.writeOverflow(" ")
			}
		}

		// draw text data view
		if app.flags.Columns["text"] {
			a.drawText(a.buffer[offset:min64(offset+bytesPerRow, size)])
		}

		// move cursor to start of new line
		app.term.writeOverflow("\r\n")
	}
}

func (a *editorArea) drawKey(key, desc string) {
	// invert foreground and background
	app.term.fg, app.term.bg = app.term.bg, app.term.fg

	// draw key
	app.term.writeOverflow(key)

	// restore foreground and background
	app.term.fg, app.term.bg = app.term.bg, app.term.fg

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
