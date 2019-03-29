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
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

type Installer interface {
	install(installableObj interfaces.IInstallable, stack interfaces.IStack, approved bool,
		renderTemplates bool, dryRun bool) error
	delete(installableObj interfaces.IInstallable, stack interfaces.IStack, approved bool,
		renderTemplates bool, dryRun bool) error
	name() string
}

// implemented installers
const MAKE = "make"

// Factory that creates installers
func New(name string, providerImpl interfaces.IProvider) (Installer, error) {
	if name == MAKE {
		return MakeInstaller{
			provider: providerImpl,
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Installer '%s' doesn't exist", name))
}

// Installs a kapp by delegating to an Installer implementation
func Install(i Installer, installableObj interfaces.IInstallable, stack interfaces.IStack, approved bool,
	renderTemplates bool, dryRun bool) error {
	log.Logger.Infof("Installing kapp '%s'...", installableObj.FullyQualifiedId())
	return i.install(installableObj, stack, approved, renderTemplates, dryRun)
}

// Deletes a kapp by delegating to an Installer implementation
func Delete(i Installer, installableObj interfaces.IInstallable, stack interfaces.IStack,
	approved bool, renderTemplates bool, dryRun bool) error {
	log.Logger.Infof("Deleting kapp '%s'...", installableObj.FullyQualifiedId())
	return i.delete(installableObj, stack, approved, renderTemplates, dryRun)
}
