package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type flags struct {
	DisplayHelp      bool
	DisplayVersion   bool
	DisplayEncodings bool
	rawColumns       string
	Columns          map[string]bool
	OffsetBase       string
	Group            int
	BytesPerRow      int
	Encoding         string
	Filename         string
}

var version = "0.0.1"

func (f *flags) init() {
	flag.BoolVar(&f.DisplayHelp, "help", false, "")
	flag.BoolVar(&f.DisplayVersion, "version", false, "")
	flag.BoolVar(&f.DisplayEncodings, "encodings", false, "")
	flag.StringVar(&f.rawColumns, "cols", "hex,text,keys", "")
	flag.StringVar(&f.OffsetBase, "offset_base", "hex", "")
	flag.IntVar(&f.Group, "group", 1, "")
	flag.IntVar(&f.BytesPerRow, "row", 16, "")
	flag.StringVar(&f.Encoding, "enc", "utf8", "")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `usage: hxe [options] [file]
valid options are:
 -help                       display this summary
 -version                    display version
 -encodings                  display a list of supported encodings for use with -enc
 -cols val                   comma-separated string of columns to display (default: "hex,text,keys")
 -offset_base dec|hex|oct    which radix to use for offsets (default: hex)
 -group                      how many bytes to display in a group (default: 1, options: 1, 2, 4, 8, 16)
 -row                        how many bytes to display per row (default: 16, options: 1-4096)
 -enc val                    which encoding to use for the textual representation of the data`)
	}
	flag.Parse()

	if f.DisplayHelp {
		flag.Usage()
		os.Exit(0)
	}

	if f.DisplayVersion {
		fmt.Printf("hxe version %s\n", version)
		os.Exit(0)
	}

	if f.DisplayEncodings {
		fmt.Fprintln(flag.CommandLine.Output(), `encodings for use with -enc:
"utf8" / "utf-8"
"cp037"
"cp1047"
"cp1140"
"cp437"
"cp850"
"cp852"
"cp855"
"cp858"
"cp860"
"cp862"
"cp863"
"cp865"
"cp866"
"iso-8859-1" / "iso8859-1"
"iso-8859-10" / "iso8859-10"
"iso-8859-13" / "iso8859-13"
"iso-8859-14" / "iso8859-14"
"iso-8859-15" / "iso8859-15"
"iso-8859-16" / "iso8859-16"
"iso-8859-2" / "iso8859-2"
"iso-8859-3" / "iso8859-3"
"iso-8859-4" / "iso8859-4"
"iso-8859-5" / "iso8859-5"
"iso-8859-6" / "iso8859-6"
"iso-8859-7" / "iso8859-7"
"iso-8859-8" / "iso8859-8"
"iso-8859-9" / "iso8859-9"
"koi8-r" / "koi8r"
"koi8-u" / "koi8u"
"macintosh"
"macintosh-cyrillic"
"windows-1250" / "windows1250"
"windows-1251" / "windows1251"
"windows-1252" / "windows1252"
"windows-1253" / "windows1253"
"windows-1254" / "windows1254"
"windows-1255" / "windows1255"
"windows-1256" / "windows1256"
"windows-1257" / "windows1257"
"windows-1258" / "windows1258"
"windows-874" / "windows874"`)
		os.Exit(0)
	}

	if f.Group != 1 && f.Group != 2 && f.Group != 4 && f.Group != 8 && f.Group != 16 {
		fmt.Fprintln(flag.CommandLine.Output(), "invalid amount of bytes per group")
		flag.Usage()
		os.Exit(1)
	}

	if f.BytesPerRow < 1 || f.BytesPerRow > 4096 || f.BytesPerRow%f.Group != 0 {
		fmt.Fprintln(flag.CommandLine.Output(), "invalid amount of bytes per row")
		flag.Usage()
		os.Exit(1)
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(flag.CommandLine.Output(), "no filename passed")
		flag.Usage()
		os.Exit(1)
	}

	f.Filename = flag.Arg(0)
	f.Encoding = strings.ToLower(f.Encoding)
	f.Columns = map[string]bool{
		"hex":  false,
		"text": false,
		"keys": false,
	}

	for _, v := range strings.Split(strings.ToLower(f.rawColumns), ",") {
		switch v {
		case "hex", "text", "keys":
			f.Columns[v] = true
		default:
			fmt.Fprintf(flag.CommandLine.Output(), "invalid column type \"%s\"\n", v)
			flag.Usage()
			os.Exit(1)
		}
	}
}
