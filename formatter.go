package pretty

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/kr/text"
)

const (
	limit = 50
)

// outputIndentLevel is covered in the SetOutputIndentLevel() func header
var outputIndentLevel = 4

// humanize is covered in the SetHumanize() function header
var humanize = false

// outputPrefixStr is covered in the SetOutputPrefixStr() function header
var outputPrefixStr = ""

// newlineAfterItems allows one to make the humanize output insert a blank
// line between entries in a somewhat sensical way... normally that is off.
var newlineAfterItems = false

// sawCloseBracketLast is incremented when a json-like '}' is seen in
// the output and then set back to 0 when anything else is seen (if one
// has 2 or 3 '}' chars in a row it'll increment til a non '}' char is
// seen... could be used to add spacing between items
var sawCloseBracketLast = 0

// currOutputLine only kicks on in 'humanize' active (set to true) mode, it
// examines all output being dumped and tracks what is on the current line
// of output and will clear that line when \n goes across the output.  This
// is used to decide if a carriage return + indent is needed when in
// human friendly output mode (if we see a ':' in the current line of output
// it means a "<key>:" header has been printed and a newline/indent is needed
// for the multi-line data to follow)
var currOutputLine = ""

type formatter struct {
	x     interface{}
	force bool
	quote bool
}

// Formatter makes a wrapper, f, that will format x as go source with line
// breaks and tabs. Object f responds to the "%v" formatting verb when both the
// "#" and " " (space) flags are set, for example:
//
//     fmt.Sprintf("%# v", Formatter(x))
//
// If one of these two flags is not set, or any other verb is used, f will
// format x according to the usual rules of package fmt.
// In particular, if x satisfies fmt.Formatter, then x.Format will be called.
func Formatter(x interface{}) (f fmt.Formatter) {
	return formatter{x: x, quote: true}
}

func (fo formatter) String() string {
	return fmt.Sprint(fo.x) // unwrap it
}

func (fo formatter) passThrough(f fmt.State, c rune) {
	s := "%"
	for i := 0; i < 128; i++ {
		if f.Flag(i) {
			s += string(i)
		}
	}
	if w, ok := f.Width(); ok {
		s += fmt.Sprintf("%d", w)
	}
	if p, ok := f.Precision(); ok {
		s += fmt.Sprintf(".%d", p)
	}
	s += string(c)
	fmt.Fprintf(f, s, fo.x)
}

// OutputIndentLevel returns the current step-wise indent that will
// be used when dumping output via pretty (defaults to 4 to start),
// see SetOutputIndentLevel() to adjust.
func OutputIndentLevel() int {
	return outputIndentLevel
}

// SetOutputIndentLevel can be used to adjust the step-wise indent for
// structure representations that are printed.  Give an integer number
// of spaces (recommended 2 or 4, default is 4 to start)
func SetOutputIndentLevel(indent int) {
	outputIndentLevel = indent
}

// Humanize will return the current true/false state of if "humanizing" of
// the output is active or not.  By default it starts off and you get what
// 'pretty' was originally set up for, a go-like structure w/details.  See
//  SetHumanize() to flip it on (and see what it does).
func Humanize() bool {
	return humanize
}

// SetHumanize can be used to flip on a more "Humanistic" form of output
// from the 'pretty' package (by default this is false and the regular
// 'pretty' package output that looks more like Go structures).  What
// "humanize" means is that the output could be used as the readable text
// output to your users.  When in humanize form 'pretty:' tags (like 'json:..'
// tags on structs) are honored for overriding field names (spaces ok) and
// supporting a form of omitempty (omitting nil intefaces/ptrs, empty arrays
// or maps, etc... but does not suppress output for false or 0 int's now).
// Anyhow, build a structure for your output, put json and pretty tags in
// the structure and then dump output easily in JSON (via json marshal) or
// dump the same struct to human friendly text for users.
func SetHumanize(b bool) {
	humanize = b
}

// NewlineAfterItems will return the current true/false state of if newlines
// after each item is desired or not.  By default this is off.
func NewlineAfterItems() bool {
	return newlineAfterItems
}

// SetNewlineAfterItems can be used to adjust humanistic output format so
// that there's an empty line between items that are printed.  By default
// it's off but this can be used to flip that on by setting to true.
func SetNewlineAfterItems(b bool) {
	newlineAfterItems = b
}

// OutputPrefixStr returns the current overall text prefix string, see
// the SetOutputPrefixStr() routine to set it.
func OutputPrefixStr() string {
	return outputPrefixStr
}

// SetOutputPrefixStr sets the output prefix string to the given string
func SetOutputPrefixStr(s string) {
	outputPrefixStr = s
}

func (fo formatter) Format(f fmt.State, c rune) {
	if fo.force || c == 'v' && f.Flag('#') && f.Flag(' ') {
		w := tabwriter.NewWriter(f, outputIndentLevel, outputIndentLevel, 1, ' ', 0)
		p := &printer{tw: w, Writer: w, visited: make(map[visit]int)}
		p.printValue(reflect.ValueOf(fo.x), true, fo.quote)
		w.Flush()
		return
	}
	fo.passThrough(f, c)
}

type printer struct {
	io.Writer
	tw      *tabwriter.Writer
	visited map[visit]int
	depth   int
}

func (p *printer) indent() *printer {
	q := *p
	q.tw = tabwriter.NewWriter(p.Writer, outputIndentLevel, outputIndentLevel, 1, ' ', 0)
	q.Writer = text.NewIndentWriter(q.tw, []byte{'\t'})
	return &q
}

func (p *printer) printInline(v reflect.Value, x interface{}, showType bool) {
	if showType && !humanize {
		writeString(p, v.Type().String())
		fmt.Fprintf(p, "(%#v)", x)
	} else {
		result := fmt.Sprintf("%#v", x)
		if humanize && result != "" && strings.TrimSpace(result) == "" {
			fmt.Fprintf(p, "\"%s\"", result)
		} else {
			fmt.Fprintf(p, "%s", result)
		}
		if humanize {
			lines := strings.Split(result, "\n")
			currOutputLine = lines[len(lines)-1]
		}
	}
}

// tagOptions is the string following a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}

// isValidTag is borrowed from Go's encoding/json
func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		default:
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
				return false
			}
		}
	}
	return true
}

// isEmptyValue determines for "humanistic" output if we want to see a given
// type or not... different than JSON in that we typically do want to see
// true or false settings, and even 0 values for various numerical types...
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// indentNeeded is called when there is an open bracket for a new structure
// or map or array to be printed, normally we always need to toss in a
// carriage return and indent *but* if we're doing humanized output we
// don't show the {}'s nor do we do the newlines', we want the items
// to appear at the very left margin and show cleanly from there
func indentNeeded() bool {
	if !humanize {
		return true
	}
	if strings.ContainsRune(currOutputLine, ':') {
		return true
	}
	return false
}

// newlineNeeded should only be used within the context of the humanized
// output mode (see callers).  It's basically deciding when a newline
// needs to be dumped ... since in humanized mode we're not dumping
// the '{' and '}' characters at all (on their own lines in normal 'pretty'
// pkg output) we need to be smarter about when newlines need to be
// printed in such situations.  This type of 'pretty' output:
// {    (normal pretty output prints the type before open bracket, which is "[]interface{}"
//   {  (this ones type is a hash with one entry with value a struct)
//     key1: {
//       structfield1: value
//       structfieldtwo: value
//	   }
//   },
//   {
//     key2: {
//       structfield1: value
//       structfieldtwo: value
//	   }
//   },
// }
// In humanized output form this comes out as:
// key1:
//   structfield1:   value
//   structfieldtwo: value
// key2:
//   structfield1:   value
//   structfieldtwo: value
// ..
// So all those newlines after the close brackets aren't needed, see
// indentNeeded() above as that handles the opening brackets and indent.
// Note that some folks may want a blank line between each entry and
// that can be done by counting the close brackets
func newlineNeeded() bool {
	if sawCloseBracketLast == 0 || (newlineAfterItems && sawCloseBracketLast == 2) {
		return true
	}
	return false
}

// printValue must keep track of already-printed pointer values to avoid
// infinite recursion.
type visit struct {
	v   uintptr
	typ reflect.Type
}

func (p *printer) printValue(v reflect.Value, showType, quote bool) {
	if p.depth > 10 {
		writeString(p, "!%v(DEPTH EXCEEDED)")
		return
	}

	var expand bool

	if humanize {
		quote = false
		showType = false
		expand = true
	}
	switch v.Kind() {
	case reflect.Bool:
		p.printInline(v, v.Bool(), showType)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.printInline(v, v.Int(), showType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p.printInline(v, v.Uint(), showType)
	case reflect.Float32, reflect.Float64:
		p.printInline(v, v.Float(), showType)
	case reflect.Complex64, reflect.Complex128:
		fmt.Fprintf(p, "%#v", v.Complex())
	case reflect.String:
		p.fmtString(v.String(), quote)
	case reflect.Map:
		t := v.Type()
		if showType {
			if !humanize {
				writeString(p, t.String())
			}
		}
		writeByte(p, '{') // '}' to balance the char
		if nonzero(v) || humanize {
			expand = !canInline(v.Type())
			pp := p
			if expand {
				if indentNeeded() {
					writeByte(p, '\n')
					pp = p.indent()
				}
			}
			keys := v.MapKeys()
			for i := 0; i < v.Len(); i++ {
				showTypeInStruct := true
				if humanize {
					showTypeInStruct = false
				}
				k := keys[i]
				mv := v.MapIndex(k)
				pp.printValue(k, false, true)
				writeByte(pp, ':')
				if expand {
					writeByte(pp, '\t')
				}
				if !humanize {
					showTypeInStruct = t.Elem().Kind() == reflect.Interface
				}
				pp.printValue(mv, showTypeInStruct, true)
				if expand {
					if humanize {
						if newlineNeeded() {
							writeString(pp, "\n")
						}
					} else {
						writeString(pp, ",\n")
					}
				} else if i < v.Len()-1 {
					writeString(pp, ", ")
				}
			}
			if expand {
				pp.tw.Flush()
			}
		}
		// '{' to balance below line
		writeByte(p, '}')
	case reflect.Struct:
		t := v.Type()
		if v.CanAddr() {
			addr := v.UnsafeAddr()
			vis := visit{addr, t}
			if vd, ok := p.visited[vis]; ok && vd < p.depth {
				p.fmtString(t.String()+"{(CYCLIC REFERENCE)}", false)
				break // don't print v again
			}
			p.visited[vis] = p.depth
		}

		if showType {
			if !humanize {
				writeString(p, t.String())
			}
		}
		writeByte(p, '{') // '}' to balance the char
		if nonzero(v) || humanize {
			expand = !canInline(v.Type())
			pp := p
			if expand {
				if indentNeeded() {
					writeByte(p, '\n')
					pp = p.indent()
				}
			}
			for i := 0; i < v.NumField(); i++ {
				showTypeInStruct := true
				if humanize {
					showTypeInStruct = false
				}
				if f := t.Field(i); f.Name != "" {
					name := f.Name
					omitEmpty := false
					if humanize {
						tag := f.Tag.Get("pretty")
						if tag == "-" {
							continue
						}
						newName, opts := parseTag(tag)
						if isValidTag(newName) {
							name = newName
						}
						omitEmpty = opts.Contains("omitempty")
						val := getField(v, i)
						if omitEmpty && isEmptyValue(val) {
							continue
						}
					}
					writeString(pp, name)
					writeByte(pp, ':')
					if expand {
						writeByte(pp, '\t')
					}
					if !humanize {
						showTypeInStruct = labelType(f.Type)
					}
				}
				pp.printValue(getField(v, i), showTypeInStruct, true)
				if humanize {
					if newlineNeeded() {
						writeByte(pp, '\n')
					}
				} else if expand {
					writeString(pp, ",\n")
				} else if i < v.NumField()-1 {
					writeString(pp, ", ")
				}
			}
			if expand {
				pp.tw.Flush()
			}
		}
		// '{' to balance below line
		writeByte(p, '}')
	case reflect.Interface:
		switch e := v.Elem(); {
		case e.Kind() == reflect.Invalid:
			writeString(p, "nil")
		case e.IsValid():
			pp := *p
			pp.depth++
			pp.printValue(e, showType, true)
		default:
			writeString(p, v.Type().String())
			writeString(p, "(nil)")
		}
	case reflect.Array, reflect.Slice:
		t := v.Type()
		if showType {
			writeString(p, t.String())
		}
		if v.Kind() == reflect.Slice && v.IsNil() && showType {
			writeString(p, "(nil)")
			break
		}
		if v.Kind() == reflect.Slice && v.IsNil() {
			writeString(p, "nil")
			break
		}
		writeByte(p, '{') // '}' to balance the char
		expand = !canInline(v.Type())
		pp := p
		if expand {
			if indentNeeded() {
				writeByte(p, '\n')
				pp = p.indent()
			}
		}
		for i := 0; i < v.Len(); i++ {
			showTypeInSlice := t.Elem().Kind() == reflect.Interface
			pp.printValue(v.Index(i), showTypeInSlice, true)
			if humanize {
				if newlineNeeded() {
					writeByte(pp, '\n')
				}
			} else if expand {
				writeString(pp, ",\n")
			} else if i < v.Len()-1 {
				writeString(pp, ", ")
			}
		}
		if expand {
			pp.tw.Flush()
		}
		// '{' to balance below line
		writeByte(p, '}')
	case reflect.Ptr:
		e := v.Elem()
		if !e.IsValid() {
			if humanize {
				writeString(p, "nil")
			} else {
				writeByte(p, '(')
				writeString(p, v.Type().String())
				writeString(p, ")(nil)")
			}
		} else {
			pp := *p
			pp.depth++
			if !humanize {
				writeByte(pp, '&')
			}
			pp.printValue(e, true, true)
		}
	case reflect.Chan:
		x := v.Pointer()
		if showType {
			writeByte(p, '(')
			writeString(p, v.Type().String())
			fmt.Fprintf(p, ")(%#v)", x)
		} else {
			fmt.Fprintf(p, "%#v", x)
		}
	case reflect.Func:
		writeString(p, v.Type().String())
		writeString(p, " {...}")
	case reflect.UnsafePointer:
		p.printInline(v, v.Pointer(), showType)
	case reflect.Invalid:
		writeString(p, "nil")
	}
}

func canInline(t reflect.Type) bool {
	if humanize {
		return false
	}
	switch t.Kind() {
	case reflect.Map:
		return !canExpand(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if canExpand(t.Field(i).Type) {
				return false
			}
		}
		return true
	case reflect.Interface:
		return false
	case reflect.Array, reflect.Slice:
		return !canExpand(t.Elem())
	case reflect.Ptr:
		return false
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return false
	}
	return true
}

func canExpand(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map, reflect.Struct,
		reflect.Interface, reflect.Array, reflect.Slice,
		reflect.Ptr:
		return true
	}
	return false
}

func labelType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface, reflect.Struct:
		return true
	}
	return false
}

func (p *printer) fmtString(s string, quote bool) {
	if quote || (humanize && s != "" && strings.TrimSpace(s) == "") {
		s = strconv.Quote(s)
	}
	writeString(p, s)
}

func tryDeepEqual(a, b interface{}) bool {
	defer func() { recover() }()
	return reflect.DeepEqual(a, b)
}

func writeByte(w io.Writer, b byte) {
	// if "humanized" output don't print struct/array format chars '{' and '}'
	// which are currently always done via writeByte only, sweet
	if humanize && (b == '{' || b == '}') {
		// '{' to balance below line, fixes dumb editor bracket matching
		if b == '}' {
			sawCloseBracketLast++
		}
		return
	}
	if humanize {
		if b == '\n' {
			currOutputLine = ""
		} else {
			currOutputLine = currOutputLine + string(b)
		}
	}
	sawCloseBracketLast = 0
	w.Write([]byte{b})
}

func writeString(w io.Writer, s string) {
	if humanize {
		// all close brackets (for fmt'ing) use writeByte() so zero it out
		if s != "" {
			sawCloseBracketLast = 0
		}
		// in case multi-line string, split it on newline, store curr last line
		lines := strings.Split(s, "\n")
		currOutputLine = lines[len(lines)-1]
	}
	io.WriteString(w, s)
}

func getField(v reflect.Value, i int) reflect.Value {
	val := v.Field(i)
	if val.Kind() == reflect.Interface && !val.IsNil() {
		val = val.Elem()
	}
	return val
}
