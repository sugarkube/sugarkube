package version

import (
	"fmt"
	"runtime"
)

// The git commit that was compiled. This will be filled in by the compiler.
var GitCommit string

// This will be populated by flags to the compiler.
var Version = ""

var BuildDate = ""

var GoVersion = runtime.Version()

var OsArch = fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
