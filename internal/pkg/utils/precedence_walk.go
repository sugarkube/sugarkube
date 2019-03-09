/*
 * Copyright 2019 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"fmt"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// A reimplementation of golang's filepath.Walk that allows indicating an order
// of preference for walking a directory tree. The `precedence` array is a list
// of prefixes that will influence the order in which files and directories are
// passed to `walkFn`. Files with prefixes earlier in the precedence array will
// be passed before files and directories with prefixes later in the precdence
// array. If a file name isn't in precedence array, it'll be passed in
// alphabetical order.
func PrecedenceWalk(root string, precedence []string, walkFn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		err = walkFn(root, nil, err)
	} else {
		err = walk(root, precedence, info, walkFn)
	}
	if err == filepath.SkipDir {
		return nil
	}
	return err
}

// Modified from the golang filepath package. Originally Copyright 2009 The Go Authors.
func walk(path string, precedence []string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	if !info.IsDir() {
		return walkFn(path, info, nil)
	}

	names, err := readDirNames(path)
	err1 := walkFn(path, info, err)
	// If err != nil, walk can't walk into this directory.
	// err1 != nil means walkFn want walk to skip this directory or stop walking.
	// Therefore, if one of err and err1 isn't nil, walk will return.
	if err != nil || err1 != nil {
		// The caller's behavior is controlled by the return value, which is decided
		// by walkFn. walkFn may ignore err and return nil.
		// If walkFn returns SkipDir, it will be handled by the caller.
		// So walk should return whatever walkFn returns.
		return err1
	}

	preferredNames := applyPrecdence(path, names, precedence)

	for _, name := range preferredNames {
		filename := filepath.Join(path, name)
		fileInfo, err := os.Lstat(filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(filename, precedence, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// Returns a list of path names sorted by precedence. File names and exact
// matches are ranked higher than directories or prefixes. Any names that don't
// match a precedence string are appended to the result array.
func applyPrecdence(rootDir string, names []string, precedence []string) []string {

	// create a map so we can group names that match precedence prefixes and
	// then apply extra logic
	matchMap := make(map[string][]string, 0)

	// build an array of all names in preferential order
	var matches []string
	var ok bool
	for _, prefix := range precedence {
		for _, name := range names {
			// append the match to an array keyed by prefix
			if strings.HasPrefix(name, prefix) {
				matches, ok = matchMap[prefix]
				if !ok {
					matches = make([]string, 0)
				}

				matches = append(matches, name)
				matchMap[prefix] = matches
			}
		}
	}

	// apply extra logic to each match - favour exact matches (by basename)
	// over partial matches, and favour files over directories
	for rule := range matchMap {
		matches := matchMap[rule]
		// the bool is true if i < j
		sort.SliceStable(matches, func(i, j int) bool {
			left := matches[i]
			right := matches[j]

			leftExtension := filepath.Ext(left)
			rightExtension := filepath.Ext(right)

			leftBaseName := strings.TrimSuffix(left, leftExtension)
			rightBaseName := strings.TrimSuffix(right, rightExtension)

			absLeft := filepath.Join(rootDir, left)
			absRight := filepath.Join(rootDir, right)

			// if both basenames match exactly, favour a file over a directory.
			// if both are files, or both are directories, sort by extension
			if leftBaseName == rule && rightBaseName == rule {
				// if only one is a file, favour it
				if isFile(absLeft) && !isFile(absRight) || !isFile(absLeft) && isFile(absRight) {
					return isFile(absLeft)
				} else if isFile(absLeft) && isFile(absRight) {
					// both are files. Return based on the extensions
					return leftExtension < rightExtension
				} else {
					// the same, so return false to cover all branches
					return false
				}
			} else if leftBaseName == rule || rightBaseName == rule {
				// if one is an exact match, favour it
				return leftBaseName == rule
			} else {
				// neither basenames are exact matches. Favour the one that matches
				// most characters
				leftStripped := strings.TrimPrefix(leftBaseName, rule)
				rightStripped := strings.TrimPrefix(rightBaseName, rule)

				if len(leftStripped) != len(rightStripped) {
					return len(leftStripped) < len(rightStripped)
				}

				// if the basenames are the same length, return alphabetically
				return strings.TrimPrefix(left, rule) < strings.Trim(right, rule)
			}
		})

		matchMap[rule] = matches
	}

	intermediateResults := make([]string, 0)
	defaultNames := make([]string, 0)

	// populate the final results array
	for _, prefix := range precedence {
		matches, ok := matchMap[prefix]
		if ok {
			intermediateResults = append(intermediateResults, matches...)
		}
	}

	// append any values for which no precedence was defined
	for _, name := range names {
		if !InStringArray(intermediateResults, name) {
			defaultNames = append(defaultNames, name)
		}
	}

	intermediateResults = append(intermediateResults, defaultNames...)

	// now perform another pass hoisting files over directories so the traversal
	// is breadth first
	files := make([]string, 0)
	dirs := make([]string, 0)

	for _, path := range intermediateResults {
		absPath := filepath.Join(rootDir, path)
		if isFile(absPath) {
			files = append(files, path)
		} else {
			dirs = append(dirs, path)
		}
	}

	results := append(files, dirs...)

	fmt.Printf("Sorted input names:\n%#v\nto preferred:\n%#v\ndefault names=%#v",
		names, results, defaultNames)

	return results
}

// Returns whether a path is a file. Panics on error
func isFile(path string) bool {
	stat, err := os.Lstat(path)
	if err != nil {
		panic(err)
	}

	if stat.IsDir() {
		log.Logger.Debugf("Path %s is a directory", path)
	} else {
		log.Logger.Debugf("Path %s is a file", path)
	}

	return !stat.IsDir()
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
// From the golang filepath package. Copyright 2009 The Go Authors.
func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
