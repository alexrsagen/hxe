package main

import "github.com/gdamore/tcell"

type areas struct {
	all     map[string]area
	current area
}

type area interface {
	onFocus() error
	onUnfocus() error
	onClose() error
	onEvent(tcell.Event) error
	init() error
}

func (f *areas) focus(name string) error {
	if f.all[name] == nil {
		return nil
	}

	// clear focus
	f.unfocus()

	// set current area to new area
	f.current = f.all[name]

	// call focus handler of new area
	if err := f.current.onFocus(); err != nil {
		return err
	}

	return nil
}

func (f *areas) unfocus() error {
	// call unfocus handler of current area
	if f.current != nil {
		if err := f.current.onUnfocus(); err != nil {
			return err
		}
	}

	// clear current area
	f.current = nil

	return nil
}

func (f *areas) add(name string, a area) error {
	f.all[name] = a
	return a.init()
}

func (f *areas) close() error {
	for _, a := range f.all {
		if err := a.onClose(); err != nil {
			return err
		}
	}
	return nil
}
