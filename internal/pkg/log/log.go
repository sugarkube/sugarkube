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
	"io/ioutil"
	"os"

	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func ConfigureLogger(logLevel string, jsonLogs bool) {
	isFirstRun := false
	if Logger == nil {
		isFirstRun = true
	} else {
		Logger.Debugf("Reconfiguring logger to log level '%s' and "+
			"setting json logs=%#v", logLevel, jsonLogs)
	}

	Logger = newLogger(logLevel, jsonLogs)

	if isFirstRun {
		Logger.Debugf("Initialised logger at log level '%s' and "+
			"json logs=%#v", logLevel, jsonLogs)
	}
}

func newLogger(logLevel string, jsonLogs bool) *logrus.Logger {
	l := logrus.New()
	l.AddHook(filename.NewHook())

	// make the formatter include the current time
	var formatter logrus.Formatter
	if jsonLogs {
		formatter = &logrus.JSONFormatter{
			DisableTimestamp: false,
		}
	} else {
		formatter = &logrus.TextFormatter{
			FullTimestamp: true,
		}
	}

	l.Formatter = formatter
	l.Out = os.Stderr

	setLevel(l, logLevel)

	return l
}

// Set the log level
func setLevel(l *logrus.Logger, level string) {
	switch level {
	case "none":
		l.Out = ioutil.Discard
	case "trace":
		l.Level = logrus.TraceLevel
	case "debug":
		l.Level = logrus.DebugLevel
	case "info":
		l.Level = logrus.InfoLevel
	case "warn":
		l.Level = logrus.WarnLevel
	case "warning":
		l.Level = logrus.WarnLevel
	case "error":
		l.Level = logrus.ErrorLevel
	case "fatal":
		l.Level = logrus.FatalLevel
	default:
		l.Level = logrus.InfoLevel
	}
}
