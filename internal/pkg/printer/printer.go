package printer

import (
	"fmt"
	"github.com/mitchellh/colorstring"
	"io"
	"os"
)

var writer io.Writer = os.Stdout

var coloriser colorstring.Colorize

func init() {
	coloriser = colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: false,
		Reset:   true,
	}
}

func Disable() {
	coloriser.Disable = true
}

func SetOutput(out io.Writer) {
	writer = out
}

// Valid colour codes are listed at: https://github.com/mitchellh/colorstring/blob/master/colorstring.go

func Fprint(text string) (int, error) {
	return fmt.Fprint(writer, coloriser.Color(text))
}

func Fprintf(format string, args ...interface{}) (int, error) {
	return fmt.Fprintf(writer, coloriser.Color(format), args...)
}

func Fprintln(text string) (int, error) {
	return fmt.Fprintln(writer, coloriser.Color(text))
}
