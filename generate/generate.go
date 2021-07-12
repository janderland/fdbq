package generate

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"text/template"
)

type InputType int

const (
	Flag InputType = iota
	EnvVar
)

// Input defines a CLI flag or environment
// variable used as input to a generator.
// An Input is usually used to populate a
// field of a struct implementing the
// Generator interface.
type Input struct {
	Type InputType

	// The where to store the input's value.
	Dst *string

	// Either the name of the flag or the name
	// of the environment variable.
	Key string
}

// Generator provides a filename and a template
// string to the Generate function. Generator
// also serves as the data provided to the
// template. See template.Template.Execute().
type Generator interface {
	Filename() string
	Template() string
}

// Generate populates the provided input, executes
// the template provided by the Generator, and writes
// the template's output to the filename provided
// by the Generator.
func Generate(gen Generator, inputs []Input) {
	for _, in := range inputs {
		in.setup()
	}
	flag.Parse()
	for _, in := range inputs {
		in.valid()
	}
	write(generate(gen))
}

func (x Input) setup() {
	switch x.Type {
	case Flag:
		flag.StringVar(x.Dst, x.Key, "", "")

	case EnvVar:
		val, exists := os.LookupEnv(x.Key)
		if !exists {
			panic(fmt.Errorf("$%s undefined", x.Key))
		}
		*x.Dst = val

	default:
		panic(fmt.Errorf("unknown input type '%v'", x.Type))
	}
}

func (x Input) valid() {
	if len(*x.Dst) > 0 {
		return
	}

	switch x.Type {
	case Flag:
		panic(fmt.Errorf("%s flag is empty", x.Key))

	case EnvVar:
		panic(fmt.Errorf("$%s is empty", x.Key))

	default:
		panic(fmt.Errorf("unknown input type '%v'", x.Type))
	}
}

func generate(gen Generator) (string, string) {
	t, err := template.New("").Parse(gen.Template())
	if err != nil {
		panic(fmt.Errorf("failed to parse template: %w", err))
	}

	var output bytes.Buffer
	if err := t.Execute(&output, gen); err != nil {
		panic(fmt.Errorf("failed to generate code: %w", err))
	}

	formatted, err := format.Source(output.Bytes())
	if err != nil {
		panic(fmt.Errorf("failed to format generated code: %w", err))
	}
	return gen.Filename(), string(formatted)
}

func write(filename, output string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(fmt.Errorf("failed to open output file: %w", err))
	}
	defer func() {
		err := file.Close()
		if err != nil {
			panic(fmt.Errorf("failed to close output file: %w", err))
		}
	}()

	_, err = fmt.Fprintln(file, output)
	if err != nil {
		panic(fmt.Errorf("failed to print output: %w", err))
	}
}
