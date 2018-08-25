package installer

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Search for files in a directory matching a regex, optionally recursively.
// If preferSymlinks is true, return paths will be replaced by symlinks where
// possible.
func findFilesByPattern(rootDir string, pattern string, recursive bool,
	preferSymlinks bool) ([]string, error) {
	re := regexp.MustCompile(pattern)
	results := make([]string, 0)

	links := make(map[string]string)

	if recursive {
		// todo - rewrite to support symlinks and excluding the .sugarkube cache directory
		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			// if the file is a symlink, save the destination so we can replace it later
			if info.Mode()&os.ModeSymlink != 0 {
				realPath, err := os.Readlink(path)
				if err != nil {
					return errors.Wrapf(err, "Error reading symlink '%s'", path)
				}

				links[realPath] = filepath.Base(path)
				return nil
			}

			if match := re.FindString(path); match != "" {
				results = append(results, path)
			}
			return nil
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if preferSymlinks && len(links) > 0 {
			// iterate through results replacing any paths that contain symlink
			// targets to be the symlinks themselves
			for i, result := range results {
				for linkTarget, link := range links {
					if strings.Contains(result, linkTarget) {
						// Too noisy. Commented out.
						//log.Debugf("Replacing link target '%s' with "+
						//	"link '%s' in result '%s'", linkTarget, link, result)
						results[i] = strings.Replace(result, linkTarget, link, 1)

						// verify that the updated path exists
						_, err := os.Stat(results[i])
						if err != nil {
							return nil, errors.Wrapf(err, "Path updated with "+
								"symlink '%s' doesn't exist", results[i])
						}
					}
				}
			}
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

	for i, result := range results {
		absResult, err := filepath.Abs(result)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		results[i] = absResult
	}

	return results, nil
}

// Returns a map of regex named capturing groups and values
func getRegExpCapturingGroups(pattern string, input string) map[string]string {
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
