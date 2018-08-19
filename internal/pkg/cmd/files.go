package cmd

import (
	"fmt"
	"strings"
)

type Files []string

func (v *Files) String() string {
	return fmt.Sprint(*v)
}

func (v *Files) Type() string {
	return "Files"
}

func (v *Files) Set(value string) error {
	for _, filePath := range strings.Split(value, ",") {
		*v = append(*v, filePath)
	}
	return nil
}
