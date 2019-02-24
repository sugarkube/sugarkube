package kapps

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"io"
)

type varsConfig struct {
	out         io.Writer
	cacheDir    string
	stackName   string
	stackFile   string
	provider    string
	provisioner string
	//kappVarsDirs cmd.Files
	profile      string
	account      string
	cluster      string
	region       string
	includeKapps []string
	excludeKapps []string
}

func newVarsCmd(out io.Writer) *cobra.Command {
	c := &varsConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "vars [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Display all variables available for a kapp"),
		Long: `Merges variables from all sources and displays them. If a kapp is given, variables available for that 
specific kapp will be displayed. If not, all generally avaialble variables for the stack will be shown.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("the name of the stack to run, and the path to the stack file are required")
			} else if len(args) > 2 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			return c.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&c.provider, "provider", "", "name of provider, e.g. aws, local, etc.")
	f.StringVar(&c.provisioner, "provisioner", "", "name of provisioner, e.g. kops, minikube, etc.")
	f.StringVar(&c.profile, "profile", "", "launch profile, e.g. dev, test, prod, etc.")
	f.StringVarP(&c.cluster, "cluster", "c", "", "name of cluster to launch, e.g. dev1, dev2, etc.")
	f.StringVarP(&c.account, "account", "a", "", "string identifier for the account to launch in (for providers that support it)")
	f.StringVarP(&c.region, "region", "r", "", "name of region (for providers that support it)")
	f.StringArrayVarP(&c.includeKapps, "include", "i", []string{}, "only process specified kapps (can specify multiple, formatted manifest-id:kapp-id)")
	f.StringArrayVarP(&c.excludeKapps, "exclude", "x", []string{}, "exclude individual kapps (can specify multiple, formatted manifest-id:kapp-id)")
	// these are commented for now to keep things simple, but ultimately we should probably support taking these as CLI args
	//f.VarP(&c.kappVarsDirs, "dir", "f", "Paths to YAML directory to load kapp values from (can specify multiple)")
	return cmd
}

func (c *varsConfig) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &kapp.StackConfig{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
		//KappVarsDirs: c.kappVarsDirs,
	}

	stackConfig, err := utils.BuildStackConfig(c.stackName, c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	candidateKapps := map[string]kapp.Kapp{}

	if len(c.includeKapps) > 0 {
		log.Logger.Debugf("Adding %d kapps to the candidate template set", len(c.includeKapps))
		candidateKapps, err = getKappsByFullyQualifiedId(c.includeKapps, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Logger.Debugf("Adding all kapps to the candidate template set")

		log.Logger.Debugf("Stack config has %d manifests", len(stackConfig.AllManifests()))

		// select all kapps
		for _, manifest := range stackConfig.AllManifests() {
			log.Logger.Debugf("Manifest '%s' contains %d kapps", manifest.Id, len(manifest.Kapps))

			for _, manifestKapp := range manifest.Kapps {
				candidateKapps[manifestKapp.FullyQualifiedId()] = manifestKapp
			}
		}
	}

	log.Logger.Debugf("There are %d candidate kapps for templating (before applying exclusions)",
		len(candidateKapps))

	if len(c.excludeKapps) > 0 {
		// delete kapps
		excludedKapps, err := getKappsByFullyQualifiedId(c.excludeKapps, stackConfig)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger.Debugf("Excluding %d kapps from the templating set", len(excludedKapps))

		for k := range excludedKapps {
			if _, ok := candidateKapps[k]; ok {
				delete(candidateKapps, k)
			}
		}
	}

	_, err = fmt.Fprintf(c.out, "Displaying variables for %d kapps:\n", len(candidateKapps))
	if err != nil {
		return errors.WithStack(err)
	}

	providerVars := stackConfig.GetProviderVars()

	for _, kappObj := range candidateKapps {
		mergedKappVars, err := kapp.MergeVarsForKapp(&kappObj, stackConfig, providerVars,
			map[string]interface{}{})

		yamlData, err := yaml.Marshal(&mergedKappVars)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = fmt.Fprintf(c.out, "\n***** Start variables for kapp '%s' *****\n"+
			"%s***** End variables for kapp '%s' *****\n",
			kappObj.Id, yamlData, kappObj.Id)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
