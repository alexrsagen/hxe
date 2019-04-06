package main

// TODO: add data editing and file save commands
// TODO: add data editor for generic data (binary, ints, floats, time, guid, disasm?)
// TODO: add data editor for defined data (file formats, structs, protobufs?)
// TODO: add simple endianness switch in data editors
// TODO: add some settings from within the editor to change flags

import (
	"os"

	"github.com/gdamore/tcell"
)

type editor struct {
	flags flags
	term  term
	areas areas

	err error // error to print after closing editor
}

var app = editor{
	areas: areas{
		all: map[string]area{},
	},
}

func main() {
	defer app.close()

	app.flags.init()
	app.term.init()
	app.must(app.areas.add("editor", &editorArea{}))
	app.areas.focus("editor")

	loop()
}

func loop() {
	for {
		// flush terminal if modified
		app.term.flush()

		ev := app.term.screen.PollEvent()
		switch v := ev.(type) {
		case *tcell.EventResize:
			// set new terminal size
			app.term.w, app.term.h = v.Size()

			// pass event to current area
			if app.areas.current != nil {
				app.must(app.areas.current.onEvent(ev))
			}

		case *tcell.EventKey:
			// handle keypress
			switch v.Key() {
			case tcell.KeyCtrlC, tcell.KeyF10:
				// close editor on C-c or F10
				return
			}

			// pass event to current area
			if app.areas.current != nil {
				app.must(app.areas.current.onEvent(ev))
			}

		default:
			// pass event to current area
			if app.areas.current != nil {
				app.must(app.areas.current.onEvent(ev))
			}
		}
	}
}

func (e *editor) close() {
	// recover any panic that happened naturally
	r := recover()

	// reset terminal so any further output will be okay
	e.term.reset()
	e.term.close()

	e.areas.close()

	// panic to dump error with stack trace
	if r != nil {
		panic(r)
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

// must checks if the last return value of a function is an error
// and panics if it is a non-nil error, otherwise returning
// the first return value.
func (e *editor) must(v ...interface{}) interface{} {
	if len(v) == 0 {
		return nil
	}
	switch x := v[len(v)-1].(type) {
	case error:
		if x != nil {
			e.error(x)
		}
	}
	return v[0]
}
