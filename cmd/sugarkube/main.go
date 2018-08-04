package main

import (
	"fmt"
	"github.com/boosh/sugarkube/internal/pkg/cmd/sugarkube"
	"os"
	"path/filepath"
)

func main() {

	baseName := filepath.Base(os.Args[0])

	if err := sugarkube.NewCommand(baseName).Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
