package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	g "github.com/janderland/fdbq/generate"
)

func main() {
	var gen VisitorGen
	g.Generate(gen, []g.Input{
		{Type: g.Flag, Dst: &gen.Visitor, Key: "visitor"},
		{Type: g.Flag, Dst: &gen.Acceptor, Key: "acceptor"},
		{Type: g.Flag, Dst: &gen.types, Key: "types"},
		{Type: g.EnvVar, Dst: &gen.Package, Key: "GOPACKAGE"},
		{Type: g.EnvVar, Dst: &gen.file, Key: "GOFILE"},
	})
}

type VisitorGen struct {
	Visitor  string
	Acceptor string
	types    string
	Package  string
	file     string
}

func (x VisitorGen) Filename() string {
	noExt := x.file[0 : len(x.file)-len(filepath.Ext(x.file))]
	return fmt.Sprintf("%s_%s.gen.go", noExt, strings.ToLower(x.Visitor))
}

func (x VisitorGen) Template() string {
	return `
// Code generated with args "{{.Args}}". DO NOT EDIT.
package {{.Package}}

type (
	{{.Visitor}} interface {
		{{range $i, $type := .Types -}}
		{{$.VisitorMethod $type}}({{$type}})
		{{end}}
	}

	{{.Acceptor}} interface {
		{{.AcceptorMethod}}({{.Visitor}})
	}
)

func _() {
	var (
		{{range .Types -}}
		{{.}}_ {{.}}
		{{end}}

		{{range .Types -}}
		_ {{$.Acceptor}} = &{{.}}_
		{{end}}
	)
}

{{range $i, $type := .Types}}
func (x *{{$type}}) {{$.AcceptorMethod}}(v {{$.Visitor}}) {
	v.{{$.VisitorMethod $type}}(*x)
}
{{end}}
`
}

func (x VisitorGen) Args() string {
	return strings.Join(os.Args[1:], " ")
}

func (x VisitorGen) Types() []string {
	return strings.Split(x.types, ",")
}

func (x VisitorGen) VisitorMethod(typ string) string {
	return "Visit" + strings.Title(typ)
}

func (x VisitorGen) AcceptorMethod() string {
	return strings.Title(x.Acceptor)
}
