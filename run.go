package main

import (
	"context"
	"fmt"
	"io"

	command "github.com/gloo-foo/cmd-perl"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

// Error is the package sentinel error type. Every error this wrapper emits is
// a const of this type, so callers can compare with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrMissingScript is returned when no perl script positional is supplied.
const ErrMissingScript Error = "no script given"

const (
	flagLoop      = "loop"
	flagPrint     = "print"
	flagAutoSplit = "autosplit"
)

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `perl [OPTIONS] SCRIPT

Run SCRIPT over standard input via perl -e.`

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags while still exposing
// the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the perl CLI against the injected version, I/O, and
// filesystem, returning the process exit code.
func run(version string, args []string, stdin io.Reader, stdout, stderr io.Writer, _ afero.Fs) int {
	cmd := newApp(version, stdin, stdout)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, "perl: %v\n", err)
		return 1
	}
	return 0
}

func newApp(version string, stdin io.Reader, stdout io.Writer) *cli.Command {
	return &cli.Command{
		Name:            "perl",
		Version:         version,
		Usage:           "perl command wrapper for yupsh",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: flagLoop, Aliases: []string{"n"}, Usage: "assume loop around script"},
			&cli.BoolFlag{Name: flagPrint, Aliases: []string{"p"}, Usage: "assume loop like -n but print line also"},
			&cli.BoolFlag{Name: flagAutoSplit, Aliases: []string{"a"}, Usage: "autosplit mode with -n or -p"},
		},
		Action: action(stdin, stdout),
	}
}

func action(stdin io.Reader, stdout io.Writer) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		script, err := scriptArg(c)
		if err != nil {
			return err
		}
		opts := append(options(c), command.PerlScript(script))
		_, err = gloo.Run(
			gloo.ByteReaderSource([]io.Reader{stdin}),
			gloo.ByteWriteTo(stdout),
			command.Perl(opts...),
		)
		return err
	}
}

func scriptArg(c *cli.Command) (string, error) {
	if c.NArg() == 0 {
		return "", ErrMissingScript
	}
	return c.Args().Get(0), nil
}

func options(c *cli.Command) []any {
	var opts []any
	if c.Bool(flagLoop) {
		opts = append(opts, command.PerlLoop)
	}
	if c.Bool(flagPrint) {
		opts = append(opts, command.PerlPrint, command.PerlLoop)
	}
	if c.Bool(flagAutoSplit) {
		opts = append(opts, command.PerlAutoSplit)
	}
	return opts
}
