package kapps

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/plan"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/program"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
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

	dagObj, err := plan.BuildDagForSelected(stackObj, c.workspaceDir, c.includeSelector, c.excludeSelector, false)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintln("")
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = printer.Fprintf("[yellow]Validating requirements for kapps...\n")
	if err != nil {
		return errors.WithStack(err)
	}

	err = Validate(stackObj, dagObj)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Validates kapps
func Validate(stackObj interfaces.IStack, dagObj *plan.Dag) error {
	numMissing := 0
	commandsSeen := make([]string, 0)

	_, err := printer.Fprintf("Validating kapps...\n")
	if err != nil {
		return errors.WithStack(err)
	}

	installables := dagObj.GetInstallables()
	for _, installable := range installables {
		descriptor := installable.GetDescriptor()

		_, err := printer.Fprintf("* [white]%s [default]requires: [white]%s\n", installable.FullyQualifiedId(),
			strings.Join(descriptor.Requires, ", "))
		if err != nil {
			return errors.WithStack(err)
		}

		// make sure required binaries exist
		err = assertBinariesExist(stackObj, installable, commandsSeen, &numMissing)
		if err != nil {
			return errors.WithStack(err)
		}

		// make sure run steps are unquely named
		err = assertUniqueRunStepNames(descriptor.RunUnits)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if numMissing > 0 {
		_, err := printer.Fprintf("\n[red]%d requirement(s) missing\n", numMissing)
		if err != nil {
			return errors.WithStack(err)
		}

		return program.SilentError{}
	} else {
		_, err := printer.Fprint("\n[green]Kapps successfully validated\n")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Returns an error if binaries for run step commands don't exist
func assertBinariesExist(stackObj interfaces.IStack, installableObj interfaces.IInstallable, commandsSeen []string,
	numMissing *int) error {
	log.Logger.Debugf("Making sure binaries exist for '%s'", installableObj.FullyQualifiedId())
	installerName := installer.RunUnit
	installerImpl, err := installer.New(installerName, stackObj.GetProvider())
	if err != nil {
		return errors.WithStack(err)
	}

	runUnitFunctions := []func(installableObj interfaces.IInstallable, stackObj interfaces.IStack, dryRun bool) ([]structs.RunStep, error){
		installerImpl.PlanInstall,
		installerImpl.ApplyInstall,
		installerImpl.PlanDelete,
		installerImpl.ApplyDelete,
		installerImpl.Clean,
		installerImpl.Output,
	}

	for _, function := range runUnitFunctions {

		// make sure all binaries declared for all run units exist
		for _, runUnit := range installableObj.GetDescriptor().RunUnits {
			for _, binary := range runUnit.Binaries {
				commandsSeen, err = assertBinaryExists(binary, commandsSeen, installableObj, numMissing)
				if err != nil {
					return errors.WithStack(err)
				}
			}
		}

		runSteps, err := function(installableObj, stackObj, true)
		if err != nil {
			return errors.WithStack(err)
		}

		for _, runStep := range runSteps {
			command := runStep.Command

			// make sure the main command for the run step exists
			commandsSeen, err = assertBinaryExists(command, commandsSeen, installableObj, numMissing)
			if err != nil {
				return errors.WithStack(err)
			}

			// make sure all binaries declared for all run steps exist
			for _, binary := range runStep.Binaries {
				commandsSeen, err = assertBinaryExists(binary, commandsSeen, installableObj, numMissing)
				if err != nil {
					return errors.WithStack(err)
				}
			}
		}
	}

	return nil
}

// Asserts that a binary exists. Returns an updated list of commands already searched for or an error
func assertBinaryExists(command string, commandsSeen []string, installableObj interfaces.IInstallable,
	numMissing *int) ([]string, error) {

	if utils.InStringArray(commandsSeen, command) {
		log.Logger.Debugf("Already searched for command '%s', won't look again", command)
		return commandsSeen, nil
	}

	commandsSeen = append(commandsSeen, command)

	path, err := exec.LookPath(command)
	if err != nil {
		_, err = printer.Fprintf("  [red]Requirement missing! Can't find '[bold]%s[reset][red]' "+
			"for [bold]%s\n", command, installableObj.FullyQualifiedId())
		*numMissing++
		if err != nil {
			return commandsSeen, errors.WithStack(err)
		}
		log.Logger.Errorf("Requirement missing. Can't find '%s' for '%s'", command, installableObj.FullyQualifiedId())
	} else {
		if config.CurrentConfig.Verbose {
			if strings.HasPrefix(command, "/") {
				_, err = printer.Fprintf("  [green]Found '[bold]%s[reset][green]'\n", command)
			} else {
				_, err = printer.Fprintf("  [green]Found '[bold]%s[reset][green]' at '%s'\n", command, path)
			}
			if err != nil {
				return commandsSeen, errors.WithStack(err)
			}
		}
		log.Logger.Infof("Found requirement '%s' at '%s'", command, path)
	}

	return commandsSeen, nil
}

// Returns an error if multiple run steps in the same run unit have the same name
func assertUniqueRunStepNames(runUnits map[string]structs.RunUnit) error {
	for _, runUnit := range runUnits {
		for _, runSteps := range [][]structs.RunStep{runUnit.PlanInstall, runUnit.ApplyInstall, runUnit.PlanDelete,
			runUnit.ApplyDelete, runUnit.Output, runUnit.Clean} {

			// strip out call run steps
			candidateRunSteps := make([]structs.RunStep, 0)
			for _, step := range runSteps {
				if step.Call == "" {
					candidateRunSteps = append(candidateRunSteps, step)
				}
			}

			err := errorOnDuplicates(candidateRunSteps)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

// Returns an error if multiple run steps have the same name
func errorOnDuplicates(runSteps []structs.RunStep) error {
	seen := make(map[string]bool, 0)
	for _, step := range runSteps {
		if _, ok := seen[step.Name]; ok {
			return fmt.Errorf("Multiple run steps exist called '%s'. Run steps in each run unit must be "+
				"uniquely named.", step.Name)
		}

		log.Logger.Tracef("No previous run step called '%s' exists", step.Name)

		seen[step.Name] = true
	}

	return nil
}
