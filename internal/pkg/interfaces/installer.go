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

package interfaces

type IInstaller interface {
	PlanInstall(installableObj IInstallable, stack IStack, dryRun bool) error
	ApplyInstall(installableObj IInstallable, stack IStack, dryRun bool) error
	PlanDelete(installableObj IInstallable, stack IStack, dryRun bool) error
	ApplyDelete(installableObj IInstallable, stack IStack, dryRun bool) error
	Clean(installableObj IInstallable, stack IStack, dryRun bool) error
	Output(installableObj IInstallable, stack IStack, dryRun bool) error
	Name() string
	GetVars(action string, approved bool) map[string]interface{}
}
