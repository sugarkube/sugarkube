package main

import (
	"github.com/boosh/sugarkube/internal/pkg/cmd"
	"github.com/boosh/sugarkube/internal/pkg/cmd/sugarkube"
	"os"
	"path/filepath"
)

func main() {

	baseName := filepath.Base(os.Args[0])

	err := sugarkube.NewCommand(baseName).Execute()
	cmd.CheckError(err)

}
