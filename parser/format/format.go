// Package format converts key-values into query strings.
package format

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/janderland/fql/keyval"
	"github.com/janderland/fql/parser/internal"
)

// Format provides methods which convert the types
// defined in keyval into strings. The methods with
// an input parameter format their input into a string
// and append the string to an internal buffer, which
// can be retrieved or cleared via the String and
// Reset methods.
type Format struct {
	builder *strings.Builder

	// When set to false, byte strings are formatted
	// as their length instead of the actual string.
	printBytes bool
}

type Option func(*Format)

func New(opts ...Option) Format {
	x := Format{
		builder: &strings.Builder{},
	}
	for _, o := range opts {
		o(&x)
	}
	return x
}

func WithPrintBytes() Option {
	return func(x *Format) {
		x.printBytes = true
	}
}

// String returns the contents of the internal buffer.
func (x *Format) String() string {
	return x.builder.String()
}

// Reset clears the contents of the internal buffer.
func (x *Format) Reset() {
	x.builder.Reset()
}

// Query formats the given keyval.Query and appends
// it to the internal buffer.
func (x *Format) Query(in keyval.Query) {
	in.Query(&formatQuery{x})
}

// KeyValue formats the given keyval.KeyValue
// and appends it to the internal buffer.
func (x *Format) KeyValue(in keyval.KeyValue) {
	x.Key(in.Key)
	x.builder.WriteRune(internal.KeyValSep)
	x.Value(in.Value)
}

// Key formats the given keyval.Key
// and appends it to the internal buffer.
func (x *Format) Key(in keyval.Key) {
	x.Directory(in.Directory)
	x.Tuple(in.Tuple)
}

// Value formats the given keyval.Value
// and appends it to the internal buffer.
func (x *Format) Value(in keyval.Value) {
	in.Value(&formatData{x})
}

// Directory formats the given keyval.Directory
// and appends it to the internal buffer.
func (x *Format) Directory(in keyval.Directory) {
	for _, element := range in {
		x.builder.WriteRune(internal.DirSep)
		element.DirElement(&formatDirElement{x})
	}
}

// Tuple formats the given keyval.Tuple
// and appends it to the internal buffer.
func (x *Format) Tuple(in keyval.Tuple) {
	x.builder.WriteRune(internal.TupStart)
	for i, element := range in {
		if i != 0 {
			x.builder.WriteRune(internal.TupSep)
		}
		element.TupElement(&formatData{x})
	}
	x.builder.WriteRune(internal.TupEnd)
}

// Variable formats the given keyval.Variable
// and appends it to the internal buffer.
func (x *Format) Variable(in keyval.Variable) {
	x.builder.WriteRune(internal.VarStart)
	for i, vType := range in {
		if i != 0 {
			x.builder.WriteRune(internal.VarSep)
		}
		x.builder.WriteString(string(vType))
	}
	x.builder.WriteRune(internal.VarEnd)
}

// Bytes formats the given keyval.Bytes
// and appends it to the internal buffer.
func (x *Format) Bytes(in keyval.Bytes) {
	if x.printBytes {
		x.builder.WriteString(internal.HexStart)
		x.builder.WriteString(hex.EncodeToString(in))
	} else {
		x.builder.WriteString(strconv.FormatInt(int64(len(in)), 10))
		x.builder.WriteString(" bytes")
	}
}

// Str formats the given keyval.String
// and appends it to the internal buffer.
func (x *Format) Str(in keyval.String) {
	x.builder.WriteRune(internal.StrMark)
	x.builder.WriteString(escapeString(string(in)))
	x.builder.WriteRune(internal.StrMark)
}

// UUID formats the given keyval.UUID
// and appends it to the internal buffer.
func (x *Format) UUID(in keyval.UUID) {
	x.builder.WriteString(hex.EncodeToString(in[:4]))
	x.builder.WriteRune('-')
	x.builder.WriteString(hex.EncodeToString(in[4:6]))
	x.builder.WriteRune('-')
	x.builder.WriteString(hex.EncodeToString(in[6:8]))
	x.builder.WriteRune('-')
	x.builder.WriteString(hex.EncodeToString(in[8:10]))
	x.builder.WriteRune('-')
	x.builder.WriteString(hex.EncodeToString(in[10:]))
}

// Bool formats the given keyval.Bool
// and appends it to the internal buffer.
func (x *Format) Bool(in keyval.Bool) {
	if in {
		x.builder.WriteString(internal.True)
	} else {
		x.builder.WriteString(internal.False)
	}
}

// Int formats the given keyval.Int
// and appends it to the internal buffer.
func (x *Format) Int(in keyval.Int) {
	x.builder.WriteString(strconv.FormatInt(int64(in), 10))
}

// Uint formats the given keyval.Uint
// and appends it to the internal buffer.
func (x *Format) Uint(in keyval.Uint) {
	x.builder.WriteString(strconv.FormatUint(uint64(in), 10))
}

// Float formats the given keyval.Float
// and appends it to the internal buffer.
func (x *Format) Float(in keyval.Float) {
	x.builder.WriteString(strconv.FormatFloat(float64(in), 'g', 10, 64))
}

// Nil formats the given keyval.Nil
// and appends it to the internal buffer.
func (x *Format) Nil(_ keyval.Nil) {
	x.builder.WriteString(internal.Nil)
}

// Clear formats the given keyval.Clear
// and appends it to the internal buffer.
func (x *Format) Clear(_ keyval.Clear) {
	x.builder.WriteString(internal.Clear)
}

// MaybeMore formats the given keyval.MaybeMore
// and appends it to the internal buffer.
func (x *Format) MaybeMore(_ keyval.MaybeMore) {
	x.builder.WriteString(internal.MaybeMore)
}

func escapeString(in string) string {
	out := strings.ReplaceAll(in, "\\", "\\\\")
	out = strings.ReplaceAll(out, "\"", "\\\"")
	return out
}
