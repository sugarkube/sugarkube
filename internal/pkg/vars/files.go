package vars

import (
	"fmt"
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
func GroupFiles(dir string) map[string][]string {
	groupedFiles := make(map[string][]string, 0)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
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
		filesForBase = append([]string{path}, filesForBase...)
		groupedFiles[baseName] = filesForBase

		return nil
	})

	if err != nil {
		log.Fatal("Error walking directory tree", dir)
	}

	return groupedFiles
}
