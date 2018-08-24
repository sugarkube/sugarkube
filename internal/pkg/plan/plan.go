package plan

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/cacher"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/provider"
	"os"
)

type Tranche struct {
	// The manifest associated with this tranche
	manifest kapp.Manifest
	// Kapps to install into the target cluster
	installables []kapp.Kapp
	// Kapps to destroy from the target cluster
	destroyables []kapp.Kapp
	// Kapps that are already in the target cluster so can be ignored
	ignorables []kapp.Kapp
}

type Plan struct {
	// installation/destruction phases. Tranches will be run sequentially, but
	// each kapp in the tranche will be processed in parallel
	tranche []Tranche
	// contains details of the target cluster
	stackConfig *kapp.StackConfig
	// a cache dir to run the (make) installer over. It should already have
	// been validated to match the stack config.
	cacheDir string
}

func Create(stackConfig *kapp.StackConfig, cacheDir string) (*Plan, error) {

	// build a plan containing all kapps, then filter out the ones that don't
	// need running based on responses from SOTs
	tranches := make([]Tranche, 0)

	for _, manifest := range stackConfig.Manifests {
		installables := make([]kapp.Kapp, 0)
		destroyables := make([]kapp.Kapp, 0)

		for _, manifestKapp := range manifest.Kapps {
			if manifestKapp.ShouldBePresent {
				installables = append(installables, manifestKapp)
			} else {
				destroyables = append(destroyables, manifestKapp)
			}
		}

		tranche := Tranche{
			manifest:     manifest,
			installables: installables,
			destroyables: destroyables,
		}

		tranches = append(tranches, tranche)
	}

	plan := Plan{
		tranche:     tranches,
		stackConfig: stackConfig,
		cacheDir:    cacheDir,
	}

	// todo - use Sources of Truth (SOTs) to discover the current set of kapps installed
	// todo - diff the cluster state with the desired state from the manifests to create a plan

	return &plan, nil
}

// Apply a plan to make a target cluster have the necessary kapps installed/
// destroyed to match the input manifests. Each tranche is run sequentially,
// and each kapp in each tranche is processed in parallel.
func (p *Plan) Apply(dryRun bool) error {

	if p.tranche == nil {
		log.Info("No tranches in plan to process")
		return nil
	}

	providerImpl, err := provider.NewProvider(p.stackConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	doneCh := make(chan bool)
	errCh := make(chan error)

	log.Debugf("Applying plan: %#v", p)

	for i, tranche := range p.tranche {
		manifestCacheDir := cacher.GetManifestCachePath(p.cacheDir, tranche.manifest)

		for _, installable := range tranche.installables {
			go processKapp(installable, p.stackConfig, manifestCacheDir, true,
				providerImpl, doneCh, errCh, dryRun)
		}

		for _, destroyable := range tranche.destroyables {
			go processKapp(destroyable, p.stackConfig, manifestCacheDir, false,
				providerImpl, doneCh, errCh, dryRun)
		}

		totalOperations := len(tranche.installables) + len(tranche.destroyables)

		for success := 0; success < totalOperations; success++ {
			select {
			case err := <-errCh:
				close(doneCh)
				log.Warnf("Error processing kapp in tranche %d of plan: %s", i+1, err)
				return errors.Wrapf(err, "Error processing kapp goroutine "+
					"in tranche %d of plan", i+1)
			case <-doneCh:
				log.Debugf("%d kapp(s) successfully processed in tranche %d",
					success+1, i+1)
			}
		}
	}

	log.Debugf("Finished applying plan")

	return nil
}

// Installs or destroys a kapp using the appropriate Installer
func processKapp(kappObj kapp.Kapp, stackConfig *kapp.StackConfig,
	manifestCacheDir string, install bool, providerImpl provider.Provider,
	doneCh chan bool, errCh chan error, dryRun bool) {

	kappRootDir := cacher.GetKappRootPath(manifestCacheDir, kappObj)

	log.Debugf("Processing kapp '%s' in %s", kappObj.Id, kappRootDir)

	_, err := os.Stat(kappRootDir)
	if err != nil {
		msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'",
			kappObj.Id, kappRootDir)
		log.Warn(msg)
		errCh <- errors.Wrap(err, msg)
	}

	// kapp exists, run the appropriate installer method
	installerImpl, err := installer.NewInstaller(installer.MAKE, providerImpl)
	if err != nil {
		errCh <- errors.Wrapf(err, "Error instantiating installer for "+
			"kapp '%s'", kappObj.Id)
	}

	// install the kapp
	if install {
		installer.Install(installerImpl, &kappObj, stackConfig, dryRun)
	} else { // destroy the kapp
		installer.Destroy(installerImpl, &kappObj, stackConfig, dryRun)
	}

	doneCh <- true
}
