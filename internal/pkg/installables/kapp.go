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

package installables

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"strings"
)

// A struct representing the raw config for a kapp that can come from YAML (but
// in future probably from other sources as well). Making a distinction between
// the raw input config and runtime values means we can stop other parts of
// the codebase grabbing values from the raw config struct directly and enforce
// that they use an interface.
type RawKappConfig struct {
	Id string
}

type Kapp struct {
	rawConfig RawKappConfig
	manifest  *kapp.Manifest
}

// Returns the non-fully qualified ID
func (k Kapp) Id() string {
	return k.rawConfig.Id
}

// Returns the fully-qualified ID of a kapp
func (k Kapp) FullyQualifiedId() string {
	if k.manifest == nil {
		return k.Id()
	} else {
		return strings.Join([]string{k.manifest.Id(), k.Id()}, constants.NamespaceSeparator)
	}
}

// Returns an array of acquirers configured for the sources for this kapp. We need to recompute these each time
// instead of caching them so that any manifest overrides will take effect.
func (k *Kapp) Acquirers() ([]acquirer.Acquirer, error) {
	acquirers, err := acquirer.GetAcquirersFromSources(k.Sources)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return acquirers, nil
}
