/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
)

func main() {

	// see https://github.com/golang/go/wiki/Performance
	cpuProfile := os.Getenv("SUGARKUBE_ENABLE_PROFILING")

	if cpuProfile != "" {
		go func() {
			addr := "localhost:6060"
			fmt.Printf("Profiling enabled on %s", addr)
			log.Println(http.ListenAndServe(addr, nil))
		}()
	}

	baseName := filepath.Base(os.Args[0])

	err := cli.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
