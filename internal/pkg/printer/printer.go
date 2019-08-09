package printer

import (
	"github.com/mitchellh/colorstring"
	"io"
	"os"
)

var writer io.Writer = os.Stdout

func SetOutput(out io.Writer) {
	writer = out
}

// Valid colour codes are listed at: https://github.com/mitchellh/colorstring/blob/master/colorstring.go

func Fprint(text string) (int, error) {
	return colorstring.Fprint(writer, text)
}

func Fprintf(format string, args ...interface{}) (int, error) {
	return colorstring.Fprintf(writer, format, args...)
}

func Fprintln(text string) (int, error) {
	return colorstring.Fprintln(writer, text)
}
