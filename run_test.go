package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		version    string
		stdin      string
		wantOut    string
		wantErrSub string
		args       []string
		wantCode   int
	}{
		{
			name:    "print uppercase each line",
			args:    []string{"perl", "-p", "$_=uc"},
			stdin:   "alpha\nbeta\n",
			wantOut: "ALPHA\nBETA\n",
		},
		{
			name:    "print substitution",
			args:    []string{"perl", "-p", "s/a/A/g"},
			stdin:   "banana\n",
			wantOut: "bAnAnA\n",
		},
		{
			name:    "loop without print",
			args:    []string{"perl", "-n", "print uc($_)"},
			stdin:   "hello\nworld\n",
			wantOut: "HELLO\nWORLD\n",
		},
		{
			name:    "print with autosplit",
			args:    []string{"perl", "-p", "-a", "$_=join(\":\",@F).\"\\n\""},
			stdin:   "one two three\n",
			wantOut: "one:two:three\n",
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"perl", "--version"},
			wantOut: "perl version 1.2.3\n",
		},
		{
			name:       "invalid script errors",
			args:       []string{"perl", "-p", "((("},
			stdin:      "x\n",
			wantCode:   1,
			wantErrSub: "perl:",
		},
		{
			name:       "missing script errors",
			args:       []string{"perl"},
			wantCode:   1,
			wantErrSub: "perl: no script given",
		},
		{
			name:       "unknown flag errors",
			args:       []string{"perl", "--nope", "$_=uc"},
			wantCode:   1,
			wantErrSub: "perl:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run(tc.version, tc.args, strings.NewReader(tc.stdin), &out, &errOut, afero.NewMemMapFs())

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub == "" && out.String() != tc.wantOut {
				t.Fatalf("stdout = %q, want %q", out.String(), tc.wantOut)
			}
			if tc.wantErrSub != "" && !strings.Contains(errOut.String(), tc.wantErrSub) {
				t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
			}
		})
	}
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
