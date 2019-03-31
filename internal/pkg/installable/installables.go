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

package installable

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
)

func New(manifestId string, descriptors []structs.KappDescriptorWithMaps) (interfaces.IInstallable, error) {

	// convert the mergedDescriptor to be a KappDescriptorWithMaps and set it as initial config layer
	kapp := &Kapp{
		manifestId:       manifestId,
		topLevelCacheDir: "",
	}

	// add each descriptor sequentially. This causes the underlying merged descriptor to be reevaluated
	for _, descriptor := range descriptors {
		err := kapp.AddDescriptor(descriptor, false)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	log.Logger.Debugf("Instantiated installable: %+v", kapp)

	return kapp, nil
}
