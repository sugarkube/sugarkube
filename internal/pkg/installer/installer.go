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
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
)

type Installer interface {
	install(kappObj *kapp.Kapp, stack interfaces.IStack, approved bool, renderTemplates bool,
		dryRun bool) error
	destroy(kappObj *kapp.Kapp, stack interfaces.IStack, approved bool, renderTemplates bool,
		dryRun bool) error
}

// implemented installers
const MAKE = "make"

// Factory that creates installers
func NewInstaller(name string, providerImpl provider.Provider) (Installer, error) {
	if name == MAKE {
		return MakeInstaller{
			provider: providerImpl,
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Installer '%s' doesn't exist", name))
}

// Installs a kapp by delegating to an Installer implementation
func Install(i Installer, kappObj *kapp.Kapp, stack interfaces.IStack,
	approved bool, renderTemplates bool, dryRun bool) error {
	log.Logger.Infof("Installing kapp '%s'...", kappObj.FullyQualifiedId())
	return i.install(kappObj, stack, approved, renderTemplates, dryRun)
}

// Destroys a kapp by delegating to an Installer implementation
func Destroy(i Installer, kappObj *kapp.Kapp, stack interfaces.IStack,
	approved bool, renderTemplates bool, dryRun bool) error {
	log.Logger.Infof("Destroying kapp '%s'...", kappObj.FullyQualifiedId())
	return i.destroy(kappObj, stack, approved, renderTemplates, dryRun)
}
