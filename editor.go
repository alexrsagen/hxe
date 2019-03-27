package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/nsf/termbox-go"
)

type editor struct {
	flags  flags
	term   term
	file   *os.File
	offset int64
	buffer []byte
	err    error
}

// general editor methods

func (e *editor) init() {
	// initialize everything needed for the editor
	e.flags.init()
	e.term.init()

	// open file
	e.file = must(os.OpenFile(e.flags.Filename, os.O_RDWR|os.O_CREATE, 0644)).(*os.File)

	// initialize buffer
	e.buffer = make([]byte, e.printableBytes())
	e.load()

	// draw content
	e.drawStatic()
	e.drawDynamic()

	// event loop
	for {
		// flush terminal if modified
		e.term.flush()

		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventResize:
			// set new terminal size
			e.term.w = ev.Width
			e.term.h = ev.Height

			// resize buffer
			if e.printableBytes() > int64(cap(e.buffer)) {
				e.buffer = make([]byte, e.printableBytes())
			}

			// redraw content
			e.term.reset()
			e.drawStatic()
			e.drawDynamic()

		case termbox.EventKey:
			// handle keypress
			switch ev.Key {
			case termbox.KeyCtrlC:
				// close editor on C-c
				e.close()
			}
		}
	}
}

func (e *editor) close() {
	// reset terminal so any further output will be okay
	e.term.reset()
	e.term.close()

	if e.file != nil {
		e.file.Close()
	}

	if e.err != nil {
		panic(e.err)
	}

	os.Exit(0)
}

func (e *editor) error(err error) {
	e.err = err
	e.close()
}

// data methods

// load fills the editor buffer from the file contents
func (e *editor) load() {
	must(e.file.ReadAt(e.buffer, e.offset))
}

// encode converts bytes from UTF-8 to the editor encoding defined in flags
func (e *editor) encode(in []byte) ([]byte, error) {
	cm, err := getCharmap(e.flags.Encoding)
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
func (e *editor) decode(in []byte) ([]byte, error) {
	cm, err := getCharmap(e.flags.Encoding)
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

func (e *editor) printableBytesPerRow() int64 {
	const offset = len("Offset(x)")
	base := e.term.w - offset
	charsPerGroup := e.flags.Group*2 + 1
	maxBytesPerRow := (base / charsPerGroup) * e.flags.Group
	return int64(min(e.flags.BytesPerRow, maxBytesPerRow))
}

func (e *editor) printableBytes() int64 {
	return e.printableBytesPerRow() * int64(e.term.h-1)
}

// drawing methods

// drawStatic draws static content to the terminal
func (e *editor) drawStatic() {
	var i, j, pad int

	// draw hex data view
	if e.flags.hasColumn("hex") {
		e.term.setCursor(pos{0, 0})
		switch e.flags.OffsetBase {
		case "hex":
			pad = (e.flags.BytesPerRow - (e.flags.BytesPerRow % 0x100)) / 0x100
			e.term.writeOverflow(strings.Repeat("\n", pad) + "Offset(h) ")
		case "dec":
			pad = (e.flags.BytesPerRow - (e.flags.BytesPerRow % 100)) / 100
			e.term.writeOverflow(strings.Repeat("\n", pad) + "Offset(d) ")
		case "oct":
			pad = (e.flags.BytesPerRow - (e.flags.BytesPerRow % 0100)) / 0100
			e.term.writeOverflow(strings.Repeat("\n", pad) + "Offset(o) ")
		}
		for i = 1; j < e.flags.BytesPerRow; i++ {
			if e.term.x >= e.term.w {
				break
			}
			if i%e.flags.Group == 0 {
				curx, curj := e.term.x, j
				switch e.flags.OffsetBase {
				case "hex":
					curpad := (curj - (curj % 0x100)) / 0x100
					for ; curpad > 0; curpad-- {
						e.term.setCursor(pos{curx, pad - curpad})
						e.term.writeOverflow("FF")
						curj -= 0xFF
					}
					e.term.setCursor(pos{curx, pad})
					e.term.writeOverflow(fmt.Sprintf("%02X", curj))
				case "dec":
					curpad := (curj - (curj % 100)) / 100
					for ; curpad > 0; curpad-- {
						e.term.setCursor(pos{curx, pad - curpad})
						e.term.writeOverflow("99")
						curj -= 99
					}
					e.term.setCursor(pos{curx, pad})
					e.term.writeOverflow(fmt.Sprintf("%02d", curj))
				case "oct":
					curpad := (curj - (curj % 0100)) / 0100
					for ; curpad > 0; curpad-- {
						e.term.setCursor(pos{curx, pad - curpad})
						e.term.writeOverflow("77")
						curj -= 077
					}
					e.term.setCursor(pos{curx, pad})
					e.term.writeOverflow(fmt.Sprintf("%02o", curj))
				}
				j += e.flags.Group
				e.term.writeOverflow(strings.Repeat(" ", e.flags.Group*2-1))
			}
		}
	}

	// draw text data view
	if e.flags.hasColumn("text") {
		e.term.writeOverflow(" Decoded text")
	}

	// move cursor to start of new line
	e.term.writeOverflow("\r\n")
}

func (e *editor) drawDynamic() {
	bytesPerRow := e.printableBytesPerRow()
	offset := e.offset
	end := offset + e.printableBytes()

	for ; offset < end; offset += bytesPerRow {
		// draw hex data view
		if e.flags.hasColumn("hex") {
			e.drawOffset(offset)
			e.drawBytes(e.buffer[offset : offset+bytesPerRow])
		}

		// draw text data view
		if e.flags.hasColumn("text") {
			e.term.writeOverflow(" ")
			e.drawText(e.buffer[offset : offset+bytesPerRow])
		}

		// move cursor to start of new line
		e.term.writeOverflow("\r\n")
	}
}

func (e *editor) drawOffset(offset int64) {
	if e.flags.hasColumn("hex") {
		switch e.flags.OffsetBase {
		case "hex":
			e.term.writeOverflow(fmt.Sprintf("%08X  ", offset))
		case "dec":
			e.term.writeOverflow(fmt.Sprintf("%08d  ", offset))
		case "oct":
			e.term.writeOverflow(fmt.Sprintf("%08o  ", offset))
		}
	}
}

func (e *editor) drawBytes(b []byte) {
	for i := 0; i < len(b); i += e.flags.Group {
		e.term.writeOverflow(strings.ToUpper(hex.EncodeToString(b[i:i+e.flags.Group])) + " ")
	}
}

func (e *editor) drawText(b []byte) {
	encoded := must(e.encode(b)).([]byte)
	for _, c := range encoded {
		if c > 31 && c < 127 {
			e.term.writeOverflow(string(c))
		} else {
			e.term.writeOverflow(".")
		}
	}
}
