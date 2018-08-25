package installer

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

// Search for files in a directory matching a regex, optionally recursively
func findFilesByPattern(rootDir string, pattern string, recursive bool) ([]string, error) {
	re := regexp.MustCompile(pattern)
	results := make([]string, 0)

	if recursive {
		// todo - rewrite to support symlinks and excluding the .sugarkube cache directory
		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			if match := re.FindString(path); match != "" {
				results = append(results, path)
			}
			return nil
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

	} else {
		files, err := ioutil.ReadDir(rootDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, f := range files {
			if match := re.FindString(f.Name()); match != "" {
				results = append(results, filepath.Join(rootDir, match))
			}
		}
	}

	return results, nil
}

// Returns a map of regex named capturing groups and values
func getRexExpCapturingGroups(pattern string, input string) map[string]string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(input)

	groups := make(map[string]string, 0)
	for i, name := range re.SubexpNames() {
		if i > 0 && i <= len(match) {
			groups[name] = match[i]
		}
	}
	return groups
}
