package app

import (
	"fmt"
	"io"
	"os"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/janderland/fql/engine"
	"github.com/janderland/fql/engine/facade"
	"github.com/janderland/fql/internal/app/fullscreen"
	"github.com/janderland/fql/internal/app/headless"
	"github.com/janderland/fql/parser/format"
)

var (
	// Version can be set via build flags and
	// defines the version printed for the
	// `-v` flag.
	Version string

	// APIVersion can be set via the build flags
	// and defines the API version FDB uses.
	APIVersion = 620

	flags *Flags
)

func init() {
	flags = SetupFlags(FQL)
}

var FQL = &cobra.Command{
	Use:     "fql [flags] query ...",
	Short:   "fql is a query language for Foundation DB",
	Version: Version,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("unexpected positional args")
		}

		log := zerolog.Nop()
		if flags.Log {
			var writer io.Writer = zerolog.ConsoleWriter{
				Out:         os.Stderr,
				FormatLevel: func(_ interface{}) string { return "" },
			}
			if flags.Fullscreen() {
				file, err := os.Create(flags.LogFile)
				if err != nil {
					return errors.Wrap(err, "failed to open logging file")
				}
				defer func() {
					if err := file.Close(); err != nil {
						fmt.Println(errors.Wrap(err, "failed to close logging file"))
					}
				}()
				writer = file
			}
			log = zerolog.New(writer).With().Timestamp().Logger()
		}

		log.Log().Str("cluster file", flags.Cluster).Msg("connecting to DB")
		if err := fdb.APIVersion(APIVersion); err != nil {
			return errors.Wrap(err, "failed to set FDB API version")
		}
		db, err := fdb.OpenDatabase(flags.Cluster)
		if err != nil {
			return errors.Wrap(err, "failed to connect to DB")
		}

		eg := engine.New(
			facade.NewTransactor(db, directory.Root()),
			engine.ByteOrder(flags.ByteOrder()),
			engine.Logger(log))

		fmt := format.New(flags.FormatOpts()...)
		out := os.Stdout

		if flags.Fullscreen() {
			app := fullscreen.App{
				Engine: eg,
				Format: fmt,
				Log:    log,
				Out:    out,

				Write:      flags.Write,
				SingleOpts: flags.SingleOpts(),
				RangeOpts:  flags.RangeOpts(),
			}
			return app.Run(cmd.Context())
		}

		app := headless.App{
			Engine: eg,
			Format: fmt,
			Out:    out,

			Write:      flags.Write,
			SingleOpts: flags.SingleOpts(),
			RangeOpts:  flags.RangeOpts(),
		}
		return app.Run(cmd.Context(), flags.Queries)
	},
}
