/*
 * Copyright 2019 The Sugarkube Authors
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

package provisioner

import (
	"fmt"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"strings"
)

// Converts an array of key-value parameters to CLI args
func parameteriseValues(args []string, valueMap map[string]string) []string {
	// In kops, booleans can only indicate truth. Passing e.g. `--bastion false`
	// trips it up. So we need to explicitly filter out such keys if they have
	// the corresponding values
	excludeBooleans := map[string]string{
		"bastion": "false",
	}

	for k, v := range valueMap {
		ignoreValue := false

		for excludeK, excludeV := range excludeBooleans {
			if k == excludeK && v == excludeV {
				log.Logger.Tracef("Ignoring kops parameter '%s' (which is '%v')",
					excludeK, excludeV)
				ignoreValue = true
			}
		}

		if ignoreValue {
			continue
		}

		key := strings.Replace(k, "_", "-", -1)

		value := fmt.Sprintf("%v", v)
		if value != "" {
			value = fmt.Sprintf("%v", v)
			// we need to separate keys & values with equals signs so kops doesn't
			// get tripped up with `--bastion true`
			args = append(args, fmt.Sprintf("--%s=%s", key, value))
		} else {
			args = append(args, "--"+key)
		}
	}

	return args
}
