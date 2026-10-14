package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestConfirm(t *testing.T) {
	const prompt = "Update forgesync 0.9.0 → 0.10.0?"

	cases := []struct {
		name string
		in   io.Reader
		want bool
	}{
		// happy path
		{name: "y accepts", in: strings.NewReader("y\n"), want: true},
		{name: "Y accepts", in: strings.NewReader("Y\n"), want: true},
		{name: "yes accepts", in: strings.NewReader("yes\n"), want: true},
		{name: "n declines", in: strings.NewReader("n\n"), want: false},
		// edge cases
		{name: "uppercase YES with surrounding spaces accepts", in: strings.NewReader("  YES  \n"), want: true},
		{name: "mixed-case Yes accepts", in: strings.NewReader("Yes\n"), want: true},
		{name: "y with a Windows line ending accepts", in: strings.NewReader("y\r\n"), want: true},
		{name: "y without a newline before EOF accepts", in: strings.NewReader("y"), want: true},
		{name: "empty line takes the N default", in: strings.NewReader("\n"), want: false},
		{name: "no declines", in: strings.NewReader("no\n"), want: false},
		{name: "yep is not yes", in: strings.NewReader("yep\n"), want: false},
		{name: "only the first line is read", in: strings.NewReader("n\ny\n"), want: false},
		// error path: an unreadable stdin never confirms
		{name: "empty stdin (EOF) declines", in: strings.NewReader(""), want: false},
		{name: "read error declines", in: failingReader{}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			got := confirm(tc.in, &out, prompt)

			if got != tc.want {
				t.Errorf("confirm() = %v, want %v", got, tc.want)
			}
			if want := prompt + " [y/N] "; out.String() != want {
				t.Errorf("prompt written = %q, want %q", out.String(), want)
			}
		})
	}
}
