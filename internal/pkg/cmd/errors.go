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

/*
Copyright 2017 the Heptio Ark contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
)

// CheckError prints err to stderr and exits with code 1 if err is not nil. Otherwise, it is a
// no-op.
func CheckError(err error) {
	if err != nil {
		if err != context.Canceled {
			var err2 error
			if log.Logger.Level == logrus.DebugLevel || log.Logger.Level == logrus.TraceLevel {
				_, err2 = fmt.Fprintf(os.Stderr, fmt.Sprintf("An error occurred: %+v\n", err))
			} else {
				_, err2 = fmt.Fprintf(os.Stderr, fmt.Sprintf("An error occurred: %v\n\n"+
					"Run with `-l debug` or `-l trace` for a full stacktrace.\n", err))
			}
			if err2 != nil {
				panic(err2)
			}
		}
		os.Exit(1)
	}
}
