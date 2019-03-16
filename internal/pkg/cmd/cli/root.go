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

package cli

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cache"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/kapps"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/version"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"strconv"
)

const longUsage = `Sugarkube is dependency management for your infrastructure. 
While its focus is Kubernetes-based clusters, it can be used to deploy your
applications onto any scriptable backend.

Dependencies are declared in 'manifest' files which describe which version of
an application to install onto whichever backend, similar to a Python/pip
'requirements.txt' file,  NPM 'package.json' or Java 'pom.xml'. Therefore 
manifests can be versioned and are fully declarative. They describe which 
versions of which applications or infrastructure should be deployed onto 
whichever clusters/backends.

Applications ("Kapps") just need to be versionable and have a Makefile with 
several standard targets to be compatible, which means if you can script it 
you can run it as a Kapp. 

Kapps should create all the infrastructure they need depending on where they're 
run. E.g. installing Chart Museum on a local Minikube cluster shouldn't create
an S3 bucket, but when it's run on AWS it should. Any infra used by more than
a single Kapp should be put into its own Kapp to simplify dependency management.

Sugarkube can also create Kubernetes clusters on various backends
(e.g. AWS, local, etc.) using a variety of provisioners (e.g. Kops, Minikube).

Use Sugarkube to:

  * Fully version your applications and infrastructure as "Kapps".
  * Automate creation and configuration of your infrastructure and kapps from 
    scratch on multiple backends to aid disaster recovery and to create 
    reproducible/ephemeral environments.
  * Automate building differently specced ephemeral dev/test environments fully 
    configured with your core dependencies (e.g. Cert Manager, Vault, etc.) so 
    you can get straight to work.
  * Push your Kapps through a sane release pipeline. Develop locally or
    on (ephemeral) dev clusters, test on staging, then release to one or 
    multiple target prod clusters. The process is up to you and Sugarkube is
    compatible with Jenkins.
  * Provide a multi-cloud and/or cloud exit strategy.
  * Split your infra/Kapps into layers. Create manifests for your core Kapps
    and for different dev teams to reflect how your organisation uses your 
    clusters. E.g. Dev Team A develop with 'core' + 'KappA', but in staging & 
    prod you run 'Core' + 'KappA' + 'KappB' + 'Monitoring'.
  * Use community Kapps to immediately install e.g. a monitoring stack with
    Prometheus, Grafana, ElasticSearch, etc. then choose which alerting 
    Kapps to install on top. Because you can layer your manifests, this 
    monitoring stack only need be deployed in particular clusters so you don't 
    bloat local/dev clusters.

Sugarkube is great for new projects, but even legacy applications can be 
migrated into Kapps. You can migrate a bit at a time to see how it helps you.

See https://sugarkube.io for more info and documentation.
`

var rootCmd = &cobra.Command{
	Short: "Sweet cluster dependency management",
	Long:  longUsage,
}

var logLevel string
var configFile string

func NewCommand(name string) *cobra.Command {

	rootCmd.Use = name

	out := rootCmd.OutOrStdout()

	rootCmd.AddCommand(
		version.NewCommand(),
		cluster.NewClusterCmds(out),
		kapps.NewKappsCmds(out),
		cache.NewCacheCmds(out),
	)

	return rootCmd
}

func init() {
	// initialise a default logger
	defaultLogLevel, ok := os.LookupEnv("SUGARKUBE_LOG_LEVEL")
	if !ok {
		defaultLogLevel = "info"
	}

	jsonLogsStr, ok := os.LookupEnv("SUGARKUBE_JSON_LOGS")
	if !ok {
		jsonLogsStr = "false"
	}

	jsonLogs, err := strconv.ParseBool(jsonLogsStr)
	if err != nil {
		jsonLogs = false
	}

	log.ConfigureLogger(defaultLogLevel, jsonLogs)

	cobra.OnInitialize(func() {
		if configFile != "" {
			log.Logger.Infof("Loading custom config file '%s'", configFile)
			config.ViperConfig.SetConfigFile(configFile)
		}
		err := config.Load(config.ViperConfig)
		if err != nil {
			cause := errors.Cause(err)
			if _, ok := cause.(viper.ConfigFileNotFoundError); ok {
				fmt.Fprintf(os.Stderr, "%s\n", cause.Error())
				log.Logger.Fatalf("No config file found. Aborting.")
			} else {
				if log.Logger.Level == logrus.DebugLevel {
					log.Logger.Fatalf("Error loading config: %+v", err)
				} else {
					log.Logger.Fatalf("Error loading config: %v\n"+
						"Run with `-l debug` for a full stacktrace.", err)
				}
			}
		}

		// reconfigure the logger based on CLI args
		log.ConfigureLogger(config.Config.LogLevel,
			config.Config.JsonLogs)
	})

	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "log level. One of none|debug|info|warn|error|fatal")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "",
		fmt.Sprintf("path to a config file. If not given, default paths "+
			"will be searched for a file called '%s.(yaml|json)'", config.ConfigFileName))
	rootCmd.PersistentFlags().BoolVarP(&jsonLogs, "json-logs", "j", false, "whether to emit JSON-formatted logs")

	// bind viper to CLI args
	bindings := map[string]string{
		"log-level": "log-level",
		"json-logs": "json-logs",
	}

	viperConfig := config.ViperConfig

	for viperKey, pFlagName := range bindings {
		err := viperConfig.BindPFlag(viperKey, rootCmd.PersistentFlags().Lookup(pFlagName))
		if err != nil {
			if log.Logger.Level == logrus.DebugLevel {
				log.Logger.Fatalf("Error binding to CLI args: %+v", err)
			} else {
				log.Logger.Fatalf("Error binding to CLI args: %v\n"+
					"Run with `-l debug` for a full stacktrace.", err)
			}
		}
	}
}
