package kapps

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os/exec"
	"strings"
)

type validateConfig struct {
	workspaceDir    string
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
}

func newValidateCmd() *cobra.Command {
	c := &validateConfig{}

	cmd := &cobra.Command{
		Use:   "validate [flags] [stack-file] [stack-name] [workspace-dir]",
		Short: fmt.Sprintf("Validate you have all the required binaries required by each kapp"),
		Long:  `Loads all kapps and makes sure the binaries they declare in their 'requires' blocks are in your path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return errors.New("some required arguments are missing")
			} else if len(args) > 3 {
				return errors.New("too many arguments supplied")
			}
			c.stackFile = args[0]
			c.stackName = args[1]
			c.workspaceDir = args[2]
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
			constants.WildcardCharacter))
	f.StringArrayVarP(&c.excludeSelector, "exclude", "x", []string{},
		fmt.Sprintf("exclude individual kapps (can specify multiple, formatted manifest-id:kapp-id or 'manifest-id:%s' for all)",
			constants.WildcardCharacter))
	return cmd
}

func (c *validateConfig) run() error {

	// CLI overrides - will be merged with any loaded from a stack config file
	cliStackConfig := &structs.StackFile{
		Provider:    c.provider,
		Provisioner: c.provisioner,
		Profile:     c.profile,
		Cluster:     c.cluster,
		Region:      c.region,
		Account:     c.account,
	}

	stackObj, err := stack.BuildStack(c.stackName, c.stackFile, cliStackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	dagObj, err := BuildDagForSelected(stackObj, c.workspaceDir, c.includeSelector, c.excludeSelector,
		false, "")
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintf("Validating requirements for kapps...\n")

	numMissing := 0

	installables := dagObj.GetInstallables()
	for _, installable := range installables {
		descriptor := installable.GetDescriptor()

		_, err := printer.Fprintf("  %s requires: %s\n", installable.FullyQualifiedId(),
			strings.Join(descriptor.Requires, ", "))
		if err != nil {
			return errors.WithStack(err)
		}

		for _, requirement := range descriptor.Requires {
			path, err := exec.LookPath(requirement)
			if err != nil {
				_, err = printer.Fprintf("  ❌ Requirement missing! Can't find '%s' for %s\n", requirement,
					installable.FullyQualifiedId())
				numMissing++
				if err != nil {
					return errors.WithStack(err)
				}
				log.Logger.Errorf("Requirement missing. Can't find: %s", requirement)
			} else {
				_, err = printer.Fprintf("  ✅ Found '%s' at '%s'\n", requirement, path)
				if err != nil {
					return errors.WithStack(err)
				}
				log.Logger.Infof("Found requirement '%s' at '%s'", requirement, path)
			}
		}
	}

	if numMissing > 0 {
		_, err = printer.Fprintf("Summary: %d requirement(s) missing\n", numMissing)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = printer.Fprint("Summary: All requirements satisfied\n")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
