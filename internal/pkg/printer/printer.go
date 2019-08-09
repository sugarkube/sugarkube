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

func Fprint(text string) (int, error) {
	return colorstring.Fprint(writer, text)
}

func Fprintf(format string, args ...interface{}) (int, error) {
	return colorstring.Fprintf(writer, format, args...)
}

func Fprintln(text string) (int, error) {
	return colorstring.Fprintln(writer, text)
}
