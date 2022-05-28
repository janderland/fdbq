package format

import (
	"encoding/hex"
	"strconv"
	"strings"

	q "github.com/janderland/fdbq/keyval"
	"github.com/janderland/fdbq/parser/internal"
)

type Cfg struct {
	// When set to false, byte strings are formatted
	// as their length instead of the actual string.
	PrintBytes bool
}

// Format provides methods which convert the types
// defined in keyval into strings. The methods with
// an input parameter format their input into a string
// and append the string to an internal buffer, which
// can be retrieved or cleared via the String and
// Reset methods.
type Format struct {
	str *strings.Builder
	cfg Cfg
}

func New(cfg Cfg) Format {
	var str strings.Builder
	return Format{str: &str, cfg: cfg}
}

// String returns the contents of the internal buffer.
func (x *Format) String() string {
	return x.str.String()
}

// Reset clears the contents of the internal buffer.
func (x *Format) Reset() {
	x.str.Reset()
}

func (x *Format) Query(in q.Query) {
	in.Query(&formatQuery{x})
}

func (x *Format) KeyValue(in q.KeyValue) {
	x.Key(in.Key)
	x.str.WriteRune(internal.KeyValSep)
	x.Value(in.Value)
}

func (x *Format) Key(in q.Key) {
	x.Directory(in.Directory)
	x.Tuple(in.Tuple)
}

func (x *Format) Value(in q.Value) {
	in.Value(&formatData{x})
}

func (x *Format) Directory(in q.Directory) {
	for _, element := range in {
		x.str.WriteRune(internal.DirSep)
		element.DirElement(&formatDirElement{x})
	}
}

func (x *Format) Tuple(in q.Tuple) {
	x.str.WriteRune(internal.TupStart)
	for i, element := range in {
		if i != 0 {
			x.str.WriteRune(internal.TupSep)
		}
		element.TupElement(&formatData{x})
	}
	x.str.WriteRune(internal.TupEnd)
}

func (x *Format) Variable(in q.Variable) {
	x.str.WriteRune(internal.VarStart)
	for i, vType := range in {
		if i != 0 {
			x.str.WriteRune(internal.VarSep)
		}
		x.str.WriteString(string(vType))
	}
	x.str.WriteRune(internal.VarEnd)
}

func (x *Format) Bytes(in q.Bytes) {
	if x.cfg.PrintBytes {
		x.str.WriteString(internal.HexStart)
		x.str.WriteString(hex.EncodeToString(in))
	} else {
		x.str.WriteString(strconv.FormatInt(int64(len(in)), 10))
		x.str.WriteString(" bytes")
	}
}

func (x *Format) Str(in q.String) {
	x.str.WriteRune(internal.StrMark)
	x.str.WriteString(string(in))
	x.str.WriteRune(internal.StrMark)
}

func (x *Format) UUID(in q.UUID) {
	x.str.WriteString(hex.EncodeToString(in[:4]))
	x.str.WriteRune('-')
	x.str.WriteString(hex.EncodeToString(in[4:6]))
	x.str.WriteRune('-')
	x.str.WriteString(hex.EncodeToString(in[6:8]))
	x.str.WriteRune('-')
	x.str.WriteString(hex.EncodeToString(in[8:10]))
	x.str.WriteRune('-')
	x.str.WriteString(hex.EncodeToString(in[10:]))
}

func (x *Format) Bool(in q.Bool) {
	if in {
		x.str.WriteString(internal.True)
	} else {
		x.str.WriteString(internal.False)
	}
}

func (x *Format) Int(in q.Int) {
	x.str.WriteString(strconv.FormatInt(int64(in), 10))
}

func (x *Format) Uint(in q.Uint) {
	x.str.WriteString(strconv.FormatUint(uint64(in), 10))
}

func (x *Format) Float(in q.Float) {
	x.str.WriteString(strconv.FormatFloat(float64(in), 'g', 10, 64))
}

func (x *Format) Nil(_ q.Nil) {
	x.str.WriteString(internal.Nil)
}

func (x *Format) Clear(_ q.Clear) {
	x.str.WriteString(internal.Clear)
}

func (x *Format) MaybeMore(_ q.MaybeMore) {
	x.str.WriteString(internal.MaybeMore)
}
