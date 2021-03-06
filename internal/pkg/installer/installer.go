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

package installer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
)

// implemented installers
const RunUnit = "run-unit"

// Factory that creates installers
func New(name string) (interfaces.IInstaller, error) {
	switch name {
	case RunUnit:
		return RunUnitInstaller{}, nil
	}

	return nil, errors.New(fmt.Sprintf("Installer '%s' doesn't exist", name))
}
