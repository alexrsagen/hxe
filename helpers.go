package main

import (
	"fmt"

	"golang.org/x/text/encoding/charmap"
)

// this file contains helper functions used in other files

// must checks if the last return value of a function is an error
// and panics if it is a non-nil error, otherwise returning
// the first return value.
func must(v ...interface{}) interface{} {
	if len(v) == 0 {
		return nil
	}
	switch x := v[len(v)-1].(type) {
	case error:
		if x != nil {
			panic(x)
		}
	}
	return v[0]
}

func getCharmap(name string) (*charmap.Charmap, error) {
	switch name {
	case "utf8", "utf-8":
		return nil, nil
	case "cp037":
		return charmap.CodePage037, nil
	case "cp1047":
		return charmap.CodePage1047, nil
	case "cp1140":
		return charmap.CodePage1140, nil
	case "cp437":
		return charmap.CodePage437, nil
	case "cp850":
		return charmap.CodePage850, nil
	case "cp852":
		return charmap.CodePage852, nil
	case "cp855":
		return charmap.CodePage855, nil
	case "cp858":
		return charmap.CodePage858, nil
	case "cp860":
		return charmap.CodePage860, nil
	case "cp862":
		return charmap.CodePage862, nil
	case "cp863":
		return charmap.CodePage863, nil
	case "cp865":
		return charmap.CodePage865, nil
	case "cp866":
		return charmap.CodePage866, nil
	case "iso-8859-1", "iso8859-1":
		return charmap.ISO8859_1, nil
	case "iso-8859-10", "iso8859-10":
		return charmap.ISO8859_10, nil
	case "iso-8859-13", "iso8859-13":
		return charmap.ISO8859_13, nil
	case "iso-8859-14", "iso8859-14":
		return charmap.ISO8859_14, nil
	case "iso-8859-15", "iso8859-15":
		return charmap.ISO8859_15, nil
	case "iso-8859-16", "iso8859-16":
		return charmap.ISO8859_16, nil
	case "iso-8859-2", "iso8859-2":
		return charmap.ISO8859_2, nil
	case "iso-8859-3", "iso8859-3":
		return charmap.ISO8859_3, nil
	case "iso-8859-4", "iso8859-4":
		return charmap.ISO8859_4, nil
	case "iso-8859-5", "iso8859-5":
		return charmap.ISO8859_5, nil
	case "iso-8859-6", "iso8859-6":
		return charmap.ISO8859_6, nil
	case "iso-8859-7", "iso8859-7":
		return charmap.ISO8859_7, nil
	case "iso-8859-8", "iso8859-8":
		return charmap.ISO8859_8, nil
	case "iso-8859-9", "iso8859-9":
		return charmap.ISO8859_9, nil
	case "koi8-r", "koi8r":
		return charmap.KOI8R, nil
	case "koi8-u", "koi8u":
		return charmap.KOI8U, nil
	case "macintosh":
		return charmap.Macintosh, nil
	case "macintosh-cyrillic":
		return charmap.MacintoshCyrillic, nil
	case "windows-1250", "windows1250":
		return charmap.Windows1250, nil
	case "windows-1251", "windows1251":
		return charmap.Windows1251, nil
	case "windows-1252", "windows1252":
		return charmap.Windows1252, nil
	case "windows-1253", "windows1253":
		return charmap.Windows1253, nil
	case "windows-1254", "windows1254":
		return charmap.Windows1254, nil
	case "windows-1255", "windows1255":
		return charmap.Windows1255, nil
	case "windows-1256", "windows1256":
		return charmap.Windows1256, nil
	case "windows-1257", "windows1257":
		return charmap.Windows1257, nil
	case "windows-1258", "windows1258":
		return charmap.Windows1258, nil
	case "windows-874", "windows874":
		return charmap.Windows874, nil
	default:
		return nil, fmt.Errorf("invalid character set \"%s\"", name)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
