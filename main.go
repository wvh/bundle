// Command bundle packs the contents of given files as variables or constants into an auto-generated go code file. It is meant for use with go:generate.
//
// The format of generated code should be compliant with gofmt.
package main

/*
duration for 100000 writes
to file:
	bufio:       76.028256ms
	unbuffered: 203.707285ms

to stdout:
	bufio:       5.706322163s
	unbuffered:  6.258759674s

conclusion: use buffered io
*/

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"go/build"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

var keywords = []string{
	"break",
	"case",
	"chan",
	"const",
	"continue",
	"default",
	"defer",
	"else",
	"fallthrough",
	"for",
	"func",
	"go",
	"goto",
	"if",
	"import",
	"interface",
	"map",
	"package",
	"range",
	"return",
	"select",
	"struct",
	"switch",
	"type",
	"var",
}

// Verbose sets the verbosity level.
var Verbose = false

// BUFSIZE determines the size of read and processing buffers.
const BUFSIZE = 4096

func quoteFile(out io.Writer, fn string) error {
	in, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer in.Close()

	buf := make([]byte, BUFSIZE)
	quoted := make([]byte, 0, 2*BUFSIZE)

	out.Write([]byte(`"`))
	for {
		n, errEOF := in.Read(buf)

		// real error
		if errEOF != nil && errEOF != io.EOF {
			return errEOF
		}

		// error was EOF, we're done
		if errEOF == io.EOF && n == 0 {
			break
		}

		quoted = strconv.AppendQuote(quoted[:0], string(buf[:n]))
		if len(quoted) > 0 {
			// we write start and end quotes ourselves, drop them
			out.Write(quoted[1 : len(quoted)-1])
		}
	}
	out.Write([]byte(`"`))

	return nil
}

func writeHeaderWithPackage(w io.Writer, name string) {
	// since Go 1.9, there's a standard header for auto-generated code: ^// Code generated .* DO NOT EDIT.$
	fmt.Fprintln(w, "// Code generated automatically; DO NOT EDIT.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "package", name)
	fmt.Fprintln(w, "")
}

func writeHeader(w io.Writer) error {
	pkgName, err := getPkgName(".")
	if err != nil {
		return err
	}
	writeHeaderWithPackage(w, pkgName)
	return nil
}

func getPkgName(dir string) (string, error) {
	pkg, err := build.Default.ImportDir(".", 0|build.IgnoreVendor)
	if err != nil {
		return "", err
	}

	return pkg.Name, nil
}

func isSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isLetter(c rune) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_'
}

func isDigit(c rune) bool {
	return '0' <= c && c <= '9'
}

func isReservedKeyword(kw string) bool {
	for _, w := range keywords {
		if kw == w {
			return true
		}
	}
	return false
}

// filterInvalidChars filters out characters that are invalid in Go variable names.
// This function is meant as a helper, not infallible valitation.
// Go lexer: https://github.com/golang/go/blob/b4e9f70412671c7ac06b4852c38a0ab82d94ddf5/src/cmd/compile/internal/gc/lex.go
func filterInvalidChars(id string) string {
	buf := []rune(id)
	l := 0

	for i, c := range buf {
		// ascii
		/*
			for isLetter(c) || isDigit(c) {
				l++
				continue
			}
		*/
		//fmt.Fprintf(os.Stderr, "char: %c\n", c)
		// unicode
		//for {
		if c >= utf8.RuneSelf {
			if unicode.IsLetter(c) || c == '_' || unicode.IsDigit(c) {
				if i == 0 && unicode.IsDigit(c) {
					// "identifier cannot begin with digit %#U"
					continue
				}
			} else {
				// "invalid identifier character %#U"
				continue
			}
		} else if isLetter(c) || isDigit(c) {
			if i == 0 && isDigit(c) {
				continue
			}

		} else {
			continue
		}

		// shift and overwrite ourselves
		if i != l {
			buf[l] = c
		}
		l++

		//}
	}
	return string(buf[:l])
}

// toCamelCase removes separator characters like dashes and underscores and sets the bordering words to CamelCase.
func toCamelCase(s string) string {
	// closure like in actual stdlib strings.Title function
	isPrevSep := true
	return strings.Map(
		func(r rune) rune {
			switch r {
			case '_', '-', ' ', ':', ',', '.':
				isPrevSep = true
				return -1
			default:
				if isPrevSep {
					isPrevSep = false
					return unicode.ToTitle(r)
				}
				isPrevSep = false
				return r
			}
		},
		s)
}

func makeVarName(name, prefix string) (string, error) {
	name = toCamelCase(name)
	name = filterInvalidChars(name)

	if prefix != "" {
		name = prefix + name
	}
	if isReservedKeyword(name) {
		return "", errors.New("reserved keyword")
	}

	return name, nil
}

func stripExtension(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func makeVarNameFromBaseName(path, prefix string) (string, error) {
	return makeVarName(stripExtension(path), prefix)
}

func makeVarNameFromFileName(path, prefix string) (string, error) {
	return makeVarName(filepath.Base(path), prefix)
}

// A Bundler holds the settings for the bundling process.
type Bundler struct {
	outFile     string
	pkgName     string
	prefix      string
	decl        string
	makeVarName func(string, string) (string, error)
}

// NewBundler initialises a new Bundler with the given settings.
func NewBundler(outFile string, pkgName string, prefix string, useConst bool, varNameFunc func(string, string) (string, error)) *Bundler {
	if varNameFunc == nil {
		varNameFunc = makeVarNameFromBaseName
	}
	return &Bundler{
		outFile:     outFile,
		prefix:      prefix,
		pkgName:     pkgName,
		makeVarName: varNameFunc,
		decl: func() string {
			if useConst {
				return "const"
			}
			return "var"
		}(),
	}
}

// ProcessFiles does the actual work by including each of the provided files into the output file.
func (bundler *Bundler) ProcessFiles(files ...string) error {
	var (
		out *os.File
		err error
	)

	if bundler.outFile != "" {
		out, err = os.Create(bundler.outFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return err
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	// use buffered writer
	bw := bufio.NewWriter(out)
	defer bw.Flush()

	// write header
	if bundler.pkgName != "" {
		writeHeaderWithPackage(bw, bundler.pkgName)
	} else {
		err = writeHeader(bw)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(bw, "// These %ss are included from files by go generate.\n", bundler.decl)
	fmt.Fprintf(bw, "%s (", bundler.decl)

	for _, fn := range files {
		if Verbose {
			fmt.Fprintln(os.Stderr, "processing file:", fn)
		}

		// get variable name
		varName, err := bundler.makeVarName(fn, bundler.prefix)
		if err != nil {
			return err
		}
		//fmt.Fprintf(bw, "%s %s = ", bundler.decl, varName)
		fmt.Fprintf(bw, "\n\t// file: %s\n", fn)
		fmt.Fprintf(bw, "\t%s = ", varName)

		err = quoteFile(bw, fn)
		if err != nil {
			return err
		}
		fmt.Fprintf(bw, "\n")
	}
	fmt.Fprintf(bw, ")\n")

	return nil
}

func main() {
	var (
		outFile  = flag.String("out", "", "`file` name to write generated code to (STDOUT if not provided)")
		prefix   = flag.String("prefix", "", "prefix for generated variables")
		useConst = flag.Bool("const", false, "use const instead of var")
		pkgName  = flag.String("pkg", "", "override package name of generated file")
		verbose  = flag.Bool("v", false, "verbose; print name of files as they are processed")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <file> <file>...\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	Verbose = *verbose

	bundler := NewBundler(*outFile, *pkgName, *prefix, *useConst, makeVarNameFromBaseName)
	err := bundler.ProcessFiles(flag.Args()...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
