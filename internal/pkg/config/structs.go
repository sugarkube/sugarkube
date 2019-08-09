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

package config

import "github.com/sugarkube/sugarkube/internal/pkg/structs"

type Config struct {
	JsonLogs   bool   `mapstructure:"json-logs"`
	NoColor    bool   `mapstructure:"no-color"` // for disabling coloured output
	LogLevel   string `mapstructure:"log-level"`
	NumWorkers int    `mapstructure:"num-workers"` // an uncontroversial name that avoids British/American spelling differences (vs 'parallelisation', etc)
	// if true, merging lists under the same map key will replace the existing list entirely. If false,
	// values from lists being merged in will be appended to the existing list
	OverwriteMergedLists bool                          `mapstructure:"overwrite-merged-lists"`
	Programs             map[string]structs.KappConfig `mapstructure:"programs"`
}
