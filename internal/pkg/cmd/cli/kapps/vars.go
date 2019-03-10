package kapps

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/utils"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	datautils "github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gopkg.in/yaml.v2"
	"io"
	"strings"
)

type varsConfig struct {
	out             io.Writer
	cacheDir        string
	stackName       string
	stackFile       string
	provider        string
	provisioner     string
	profile         string
	account         string
	cluster         string
	region          string
	includeSelector []string
	excludeSelector []string
	suppress        []string
}

func newVarsCmd(out io.Writer) *cobra.Command {
	c := &varsConfig{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "vars [flags] [stack-file] [stack-name]",
		Short: fmt.Sprintf("Display all variables available for a kapp"),
		Long: `Merges variables from all sources and displays them. If a kapp is given, variables available for that 
specific kapp will be displayed. If not, all generally available variables for the stack will be shown.`,
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
	f.StringArrayVarP(&c.includeSelector, "include", "i", []string{},
		fmt.Sprintf("only process specified kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			kapp.WildcardCharacter))
	f.StringArrayVarP(&c.excludeSelector, "exclude", "x", []string{},
		fmt.Sprintf("exclude individual kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			kapp.WildcardCharacter))
	f.StringArrayVarP(&c.suppress, "suppress", "s", []string{},
		"paths to variables to suppress from the output to simplify it (e.g. 'provision.specs')")
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
	}

	stackConfig, err := utils.BuildStackConfig(c.stackName, c.stackFile, cliStackConfig, c.out)
	if err != nil {
		return errors.WithStack(err)
	}

	selectedKapps, err := kapp.SelectKapps(stackConfig.Manifests, c.includeSelector, c.excludeSelector)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = fmt.Fprintf(c.out, "Displaying variables for %d kapps:\n", len(selectedKapps))
	if err != nil {
		return errors.WithStack(err)
	}

	for _, kappObj := range selectedKapps {
		templatedVars, err := stackConfig.TemplatedVars(&kappObj, map[string]interface{}{})
		if err != nil {
			return errors.WithStack(err)
		}

		if len(c.suppress) > 0 {
			for _, exclusion := range c.suppress {
				// trim any leading zeroes for compatibility with how variables are referred to in templates
				exclusion = strings.TrimPrefix(exclusion, ".")
				blanked := datautils.BlankNestedMap(map[string]interface{}{}, strings.Split(exclusion, "."))
				log.Logger.Debugf("blanked=%#v", blanked)

				err = mergo.Merge(&templatedVars, blanked, mergo.WithOverride)
				if err != nil {
					return errors.WithStack(err)
				}
			}
		}

		yamlData, err := yaml.Marshal(&templatedVars)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = fmt.Fprintf(c.out, "\n***** Start variables for kapp '%s' *****\n"+
			"%s***** End variables for kapp '%s' *****\n",
			kappObj.FullyQualifiedId(), yamlData, kappObj.FullyQualifiedId())
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
