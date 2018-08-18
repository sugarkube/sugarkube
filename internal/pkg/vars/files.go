package vars

import (
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path/filepath"
)

// Walks a dir tree and returns a map where keys are the basename of each
// file, and values are paths to each file with that basename in th etree.
//
// E.g. given a tree:
//  |--values.yaml
//     |--subdir1
//        |--values.yaml
//     |--subdir2
//        |--values.yaml
//        |--more-values.yaml
//
// this will return:
//
// {
//	"values.yaml": []string{
//			"values.yaml",
//			"./subdir1/values.yaml",
//			"./subdir2/values.yaml",
//	},
//	"more-values.yaml": []string{
//			"./subdir2/more-values.yaml",
//	},
// }
//
func GroupFiles(dir string) (map[string][]string, error) {
	groupedFiles := make(map[string][]string, 0)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// todo - raise an error; structured logging?
			log.Warnf("Error walking dir tree %s: %v", dir, err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		baseName := info.Name()

		filesForBase, ok := groupedFiles[baseName]
		if !ok {
			filesForBase = make([]string, 0)
		}

		// prepend to the array
		filesForBase = append([]string{filepath.Clean(path)}, filesForBase...)
		groupedFiles[baseName] = filesForBase

		return nil
	})

	if err != nil {
		// todo - raise an error; structured logging?
		log.Warnf("Error walking directory tree %s", dir)
		return nil, errors.Wrapf(err, "Error walking directory tree %s", dir)
	}

	log.Debugf("Grouped files: %#v", groupedFiles)

	return groupedFiles, nil
}
