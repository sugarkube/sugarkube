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

package log

import (
	"os"

	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
)

var Logger *logrus.Logger

func init() {
	Logger = NewLogger(config.Config())
}

// todo - remove the dependency on the config object so we can log in the config
//  module (i.e. remove the circular dependency)
func NewLogger(cfg config.Provider) *logrus.Logger {
	l := logrus.New()
	l.AddHook(filename.NewHook())

	if cfg.GetBool("json_logs") {
		l.Formatter = new(logrus.JSONFormatter)
	}
	l.Out = os.Stderr

	SetLevel(l, cfg.GetString("loglevel"))

	return l
}

func SetLevel(l *logrus.Logger, level string) {
	switch level {
	case "debug":
		l.Level = logrus.DebugLevel
	case "info":
		l.Level = logrus.InfoLevel
	case "warn":
		l.Level = logrus.WarnLevel
	case "warning":
		l.Level = logrus.WarnLevel
	default:
		l.Level = logrus.InfoLevel
	}
}
