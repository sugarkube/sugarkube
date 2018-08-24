package plan

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

type Tranche struct {
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
	stackConfig kapp.StackConfig
	// a cache dir to run the (make) installer over. It should already have
	// been validated to match the stack config.
	cacheDir string
}

func Create(stackConfig *kapp.StackConfig) (*Plan, error) {
	// todo - use Sources of Truth (SOTs) to discover the current set of kapps installed
	// todo - diff the cluster state with the desired state from the manifests to create a plan
	return nil, nil
}

// Apply a plan to make a target cluster have the necessary kapps installed/
// destroyed to match the input manifests. Each tranche is run sequentially,
// and each kapp in each tranche is processed in parallel.
func Apply(plan *Plan, dryRun bool) error {
	doneCh := make(chan bool)
	errCh := make(chan error)

	log.Debugf("Applying plan: %#v", plan)

	for i, tranche := range plan.tranche {
		for _, trancheKapp := range tranche.installables {
			go processKapp(trancheKapp, doneCh, errCh, dryRun)
		}

		for _, trancheKapp := range tranche.destroyables {
			go processKapp(trancheKapp, doneCh, errCh, dryRun)
		}

		totalOperations := len(tranche.installables) + len(tranche.destroyables)

		for success := 0; success < totalOperations; success++ {
			select {
			case err := <-errCh:
				close(doneCh)
				log.Warnf("Error processing kapp in tranche %d of plan: %s", i, err)
				return errors.Wrapf(err, "Error processing kapp goroutine "+
					"in tranche %d of plan", i)
			case <-doneCh:
				log.Debugf("%d kapp(s) successfully processed in tranche %d",
					success+1, i)
			}
		}
	}

	log.Debugf("Finished applying plan")

	return nil
}

// Installs or destroys a kapp using the appropriate Installer
func processKapp(kapp kapp.Kapp, doneCh chan bool, errCh chan error, dryRun bool) {

	log.Debugf("Would process kapp: %s", kapp.Id)

	// todo - finish
	//acquirerId, err := a.Id()
	//if err != nil {
	//	errCh <- errors.Wrap(err, "Invalid acquirer ID")
	//}
	//
	//sourceDest := filepath.Join(kappCacheDir, acquirerId)
	//
	//if dryRun {
	//	log.Debugf("Dry run: Would acquire source into: %s", sourceDest)
	//} else {
	//	err := a.Acquire(sourceDest)
	//	if err != nil {
	//		errCh <- errors.WithStack(err)
	//	}
	//}
	//
	//if dryRun {
	//	log.Debugf("Dry run. Would symlink cached source %s to %s", sourcePath, symLinkTarget)
	//} else {
	//	if _, err := os.Stat(filepath.Join(kappDir, sourcePath)); err != nil {
	//		errCh <- errors.Wrapf(err, "Symlink source '%s' doesn't exist", sourcePath)
	//	}
	//
	//	log.Debugf("Symlinking cached source %s to %s", sourcePath, symLinkTarget)
	//	err := os.Symlink(sourcePath, symLinkTarget)
	//	if err != nil {
	//		errCh <- errors.Wrapf(err, "Error symlinking kapp source")
	//	}
	//}

	doneCh <- true
}
