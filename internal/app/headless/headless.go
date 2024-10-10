package headless

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/janderland/fql/engine"
	q "github.com/janderland/fql/keyval"
	"github.com/janderland/fql/keyval/class"
	"github.com/janderland/fql/keyval/convert"
	"github.com/janderland/fql/parser"
	"github.com/janderland/fql/parser/format"
	"github.com/janderland/fql/parser/scanner"
)

type App struct {
	Engine engine.Engine
	Format format.Format
	Out    io.Writer

	Write      bool
	SingleOpts engine.SingleOpts
	RangeOpts  engine.RangeOpts
}

func (x *App) Run(ctx context.Context, queries []string) error {
	_, err := x.Engine.Transact(func(eg engine.Engine) (interface{}, error) {
		for _, str := range queries {
			p := parser.New(scanner.New(strings.NewReader(str)))
			query, err := p.Parse()
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse query")
			}

			if dir, ok := query.(q.Directory); ok {
				if err := x.directories(ctx, eg, dir); err != nil {
					return nil, err
				}
				continue
			}

			var kv q.KeyValue
			if key, ok := query.(q.Key); ok {
				kv = q.KeyValue{Key: key, Value: q.Variable{}}
			} else {
				kv = query.(q.KeyValue)
			}

			switch c := class.Classify(kv); c {
			case class.Constant:
				if err := x.set(eg, kv); err != nil {
					return nil, errors.Wrap(err, "failed to execute as set query")
				}

			case class.Clear:
				if err := x.clear(eg, kv); err != nil {
					return nil, errors.Wrap(err, "failed to execute as clear query")
				}

			case class.ReadSingle:
				if err := x.singleRead(eg, kv); err != nil {
					return nil, errors.Wrap(err, "failed to execute as single read query")
				}

			case class.ReadRange:
				if err := x.rangeRead(ctx, eg, kv); err != nil {
					return nil, errors.Wrap(err, "failed to execute as range read query")
				}

			default:
				return nil, errors.Errorf("unexpected query class '%v'", c)
			}
		}
		return nil, nil
	})
	return err
}

func (x *App) set(eg engine.Engine, query q.KeyValue) error {
	if !x.Write {
		return errors.New("writing isn't enabled")
	}
	return eg.Set(query)
}

func (x *App) clear(eg engine.Engine, query q.KeyValue) error {
	if !x.Write {
		return errors.New("writing isn't enabled")
	}
	return eg.Clear(query)
}

func (x *App) singleRead(eg engine.Engine, query q.KeyValue) error {
	kv, err := eg.ReadSingle(query, x.SingleOpts)
	if err != nil {
		return err
	}
	if kv == nil {
		return nil
	}

	x.Format.Reset()
	x.Format.KeyValue(*kv)
	if _, err := fmt.Fprintln(x.Out, x.Format.String()); err != nil {
		return errors.Wrap(err, "failed to print output")
	}
	return nil
}

func (x *App) rangeRead(ctx context.Context, eg engine.Engine, query q.KeyValue) error {
	for kv := range eg.ReadRange(ctx, query, x.RangeOpts) {
		if kv.Err != nil {
			return kv.Err
		}

		x.Format.Reset()
		x.Format.KeyValue(kv.KV)
		if _, err := fmt.Fprintln(x.Out, x.Format.String()); err != nil {
			return errors.Wrap(err, "failed to print output")
		}
	}
	return nil
}

func (x *App) directories(ctx context.Context, eg engine.Engine, query q.Directory) error {
	for dir := range eg.Directories(ctx, query) {
		if dir.Err != nil {
			return dir.Err
		}

		x.Format.Reset()
		x.Format.Directory(convert.FromStringArray(dir.Dir.GetPath()))
		if _, err := fmt.Fprintln(x.Out, x.Format.String()); err != nil {
			return errors.Wrap(err, "failed to print output")
		}
	}
	return nil
}
