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
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"os/user"
	"path"
	"strings"
)

const ConfigFileName = "sugarkube-conf"

var CurrentConfig *Config
var ViperConfig *viper.Viper

func init() {
	ViperConfig = initViper("SUGARKUBE")
}

func initViper(appName string) *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix(appName)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Viper currently lowercases all keys in config files it loads. This breaks
	// defining env vars in our config file, so storing program configs in
	// our config file is on hold until this or similar are merged:
	// https://github.com/spf13/viper/pull/635
	//v.SetKeyCaseSensitivity(true)

	// global defaults
	v.SetDefault("json_logs", false)
	v.SetDefault("log_level", "info")
	v.SetDefault("num_workers", "5")
	v.SetDefault("overwrite_merged_lists", false)

	v.SetConfigName(ConfigFileName)

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
	var newConfig *Config

	err := viperConfig.ReadInConfig()
	if err != nil {
		return errors.Wrapf(err, "Error loading configuration")
	}

	err = viperConfig.Unmarshal(&newConfig)
	if err != nil {
		return errors.Wrapf(err, "Error unmarshalling config")
	}

	log.Logger.Debugf("Loaded config struct: %#v", newConfig)

	CurrentConfig = newConfig

	return nil
}
