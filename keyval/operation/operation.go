/*
Operation generates the interfaces and glue-code for the visitor pattern.
The visitor pattern is used in place of tagged-unions in this codebase
since Go does not support that feature.

Instead of implementing a type switch which handles every type in the
union, a visitor interface is implemented with methods handling each
type. The correct method is called at runtime via some generated glue
code. This glue code also defines an interface for the union itself,
allowing us to avoid using interface{}.

Structs implementing these visitor interfaces define a parameterized
(generic) function for the types in the union. For this reason, they
are called "operations" rather than "visitors" in this codebase.

Usage:

	//go:generate go run ./operation [flags]

Flags:

	-op-name string      the name of the interface of the visitor
	-param-name string   the name of the interface being visited
	-types string        a comma-separated list of concrete types to visit
*/
package main

import (
	"strings"
	"text/template"
	"unicode"

	g "github.com/janderland/fql/internal/generate"
)

func main() {
	var gen operationGen
	g.Generate(&gen, []g.Input{
		{Type: g.Flag, Dst: &gen.opName, Key: "op-name"},
		{Type: g.Flag, Dst: &gen.paramName, Key: "param-name"},
		{Type: g.Flag, Dst: &gen.types, Key: "types"},
	})
}

type operationGen struct {
	opName    string
	paramName string
	types     string
}

func (x operationGen) Name() string {
	return x.opName
}

func (x operationGen) Data() interface{} {
	return x
}

func (x operationGen) Template() *template.Template {
	return template.Must(template.New("").Parse(`
type (
	{{.OpName}} interface {
		{{range $i, $type := .Types -}}
		// {{$.VisitorMethod $type}} performs the {{$.OpName}} if the given {{$.ParamName}} is of type {{$type}}.
		{{$.VisitorMethod $type}}({{$type}})
		{{end}}
	}

	{{.ParamName}} interface {
		// {{.AcceptorMethod}} executes the given {{.OpName}} on this {{.ParamName}}.
		{{.AcceptorMethod}}({{.OpName}})

		// Eq returns true if the given value is equal to this {{.ParamName}}.
		Eq(interface{}) bool
	}
)

func _() {
	var (
		{{range .Types -}}
		{{.}} {{.}}
		{{end}}

		{{range .Types -}}
		_ {{$.ParamName}} = &{{.}}
		{{end}}
	)
}

{{range $i, $type := .Types}}
func (x {{$type}}) {{$.AcceptorMethod}}(op {{$.OpName}}) {
	op.{{$.VisitorMethod $type}}(x)
}
{{end}}
`))
}

func (x operationGen) OpName() string {
	return x.opName + "Operation"
}

func (x operationGen) ParamName() string {
	return x.paramName
}

func (x operationGen) Types() []string {
	return strings.Split(x.types, ",")
}

func (x operationGen) VisitorMethod(typ string) string {
	if len(typ) > 0 {
		typ = string(unicode.ToUpper(rune(typ[0]))) + typ[1:]
	}
	return "For" + typ
}

func (x operationGen) AcceptorMethod() string {
	if len(x.paramName) == 0 {
		return ""
	}
	return string(unicode.ToUpper(rune(x.paramName[0]))) + x.paramName[1:]
}
