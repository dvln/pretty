// Package pretty provides pretty-printing for Go values. This is
// useful during debugging, to avoid wrapping long output lines in
// the terminal.
//
// It provides a function, Formatter, that can be used with any
// function that accepts a format string. It also provides
// convenience wrappers for functions in packages fmt and log.
package pretty

import (
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/dvln/text"
)

// Errorf is a convenience wrapper for fmt.Errorf.
//
// Calling Errorf(f, x, y) is equivalent to
// fmt.Errorf(f, Formatter(x), Formatter(y)).
func Errorf(format string, a ...interface{}) error {
	str := text.Indent(fmt.Sprintf(format, wrap(a, false)...), outputPrefixStr)
	return fmt.Errorf("%s", str)
}

// Fprintf is a convenience wrapper for fmt.Fprintf.
//
// Calling Fprintf(w, f, x, y) is equivalent to
// fmt.Fprintf(w, f, Formatter(x), Formatter(y)).
func Fprintf(w io.Writer, format string, a ...interface{}) (n int, error error) {
	str := text.Indent(fmt.Sprintf(format, wrap(a, false)...), outputPrefixStr)
	return fmt.Fprint(w, str)
}

// Log is a convenience wrapper for log.Printf.
//
// Calling Log(x, y) is equivalent to
// log.Print(Formatter(x), Formatter(y)), but each operand is
// formatted with "%# v".
func Log(a ...interface{}) {
	str := text.Indent(fmt.Sprint(wrap(a, true)...), outputPrefixStr)
	log.Print(str)
}

// Logf is a convenience wrapper for log.Printf.
//
// Calling Logf(f, x, y) is equivalent to
// log.Printf(f, Formatter(x), Formatter(y)).
func Logf(format string, a ...interface{}) {
	str := text.Indent(fmt.Sprintf(format, wrap(a, false)...), outputPrefixStr)
	log.Print(str)
}

// Logln is a convenience wrapper for log.Printf.
//
// Calling Logln(x, y) is equivalent to
// log.Println(Formatter(x), Formatter(y)), but each operand is
// formatted with "%# v".
func Logln(a ...interface{}) {
	str := text.Indent(fmt.Sprintln(wrap(a, true)...), outputPrefixStr)
	log.Print(str)
}

// Print pretty-prints its operands and writes to standard output.
//
// Calling Print(x, y) is equivalent to
// fmt.Print(Formatter(x), Formatter(y)), but each operand is
// formatted with "%# v".
func Print(a ...interface{}) (n int, errno error) {
	str := text.Indent(fmt.Sprint(wrap(a, true)...), outputPrefixStr)
	return fmt.Print(str)
}

// Printf is a convenience wrapper for fmt.Printf.
//
// Calling Printf(f, x, y) is equivalent to
// fmt.Printf(f, Formatter(x), Formatter(y)).
func Printf(format string, a ...interface{}) (n int, errno error) {
	str := text.Indent(fmt.Sprintf(format, wrap(a, false)...), outputPrefixStr)
	return fmt.Print(str)
}

// Println pretty-prints its operands and writes to standard output.
//
// Calling Print(x, y) is equivalent to
// fmt.Println(Formatter(x), Formatter(y)), but each operand is
// formatted with "%# v".
func Println(a ...interface{}) (n int, errno error) {
	str := text.Indent(fmt.Sprintln(wrap(a, true)...), outputPrefixStr)
	return fmt.Println(str)
}

// Sprint is a convenience wrapper for fmt.Sprintf.
//
// Calling Sprint(x, y) is equivalent to
// fmt.Sprint(Formatter(x), Formatter(y)), but each operand is
// formatted with "%# v".
func Sprint(a ...interface{}) string {
	return fmt.Sprint(wrap(a, true)...)
}

// Sprintf is a convenience wrapper for fmt.Sprintf.
//
// Calling Sprintf(f, x, y) is equivalent to
// fmt.Sprintf(f, Formatter(x), Formatter(y)).
func Sprintf(format string, a ...interface{}) string {
	return text.Indent(fmt.Sprintf(format, wrap(a, false)...), outputPrefixStr)
}

// Sprintln is a convenience wrapper for fmt.Sprintln.
//
// Calling Sprintln(x, y) is equivalent to
// fmt.Sprintln(Formatter(x), Formatter(y)).
func Sprintln(a ...interface{}) string {
	return text.Indent(fmt.Sprintln(wrap(a, false)...), outputPrefixStr)
}

func wrap(a []interface{}, force bool) []interface{} {
	w := make([]interface{}, len(a))
	for i, x := range a {
		w[i] = formatter{v: reflect.ValueOf(x), force: force}
	}
	return w
}
