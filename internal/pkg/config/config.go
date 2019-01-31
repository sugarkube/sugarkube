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

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"os"
	"os/user"
	"path"
	"strings"
)

var Config *Conf
var ViperConfig *viper.Viper

func init() {
	ViperConfig = initViper("SUGARKUBE")
}

func initViper(appName string) *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix(appName)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// global defaults
	v.SetDefault("json-logs", false)
	v.SetDefault("log-level", "info")

	v.SetConfigName("sugarkube")

	// add look-up paths (from highest priority to lowest)
	// current working directory
	cwd, err := os.Getwd()
	if err == nil {
		v.AddConfigPath(cwd)
	}

	// user's home dir (if we can retrieve it)
	usr, err := user.Current()
	if err == nil {
		v.AddConfigPath(path.Join(usr.HomeDir, ".sugarkube"))
	}

	v.AddConfigPath("/etc/sugarkube")

	// add the directory containing this binary
	v.AddConfigPath(".")

	return v
}

// Load/Reload the configuration
func Load(viperConfig *viper.Viper) error {
	var newConf *Conf

	err := viperConfig.ReadInConfig()
	if err != nil {
		return errors.Wrapf(err, "Error loading configuration")
	}

	err = viperConfig.Unmarshal(&newConf)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling config")
	}

	Config = newConf

	return nil
}
