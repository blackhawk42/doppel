package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sanitizeDir takes the path of a directory, gets its absolute path, and makes
// sure it exists and it's, in fact, a directory.
func sanitizeDir(dirPath string) (string, error) {
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return dirPath, fmt.Errorf("error while getting absolute path of %s: %v", dirPath, err)
	}

	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		return dirPath, fmt.Errorf("error while getting infromation of %s: %v", dirPath, err)
	}

	if !dirInfo.IsDir() {
		return dirPath, fmt.Errorf("error: %s is not a directory", dirPath)
	}

	return dirPath, nil
}

// isParentDirectory takes two directories and checks if the first directory
// is a parent of the second directory.
func isParentDir(possibleParent, suspectedChild string) bool {
	// TODO: make the test something more robust that simple string prefix checking
	return strings.HasPrefix(suspectedChild, possibleParent)
}

// normilizeDirs takes a slice of directory names and returns another slice with
// only the most general directories in the hierarchy. In other words, it eliminates
// all directories that can be considered a "child" of another, leaving only the
// parents. It also removes repeated names.
//
// Only tested with full paths, no relative ones. Should pass sanitizeDir() to
// each element beforehand.
func normalizeDirs(dirPaths []string) []string {
	results := make([]string, 0, len(dirPaths))

	// Iterate all given directories
	for _, dir := range dirPaths {
		hadParent := false

		// For each directory, iterate all current results
		for i, resultDir := range results {

			// If we find a parent in the results, mark it and stop search
			if isParentDir(resultDir, dir) {
				hadParent = true
				break
			}

			// If we find that the current dir is a parent of a result, the current,
			// more general dir trumps the result and replaces it. We also mark
			// and break
			if isParentDir(dir, resultDir) {
				results[i] = dir
				hadParent = true
				break
			}
		}

		// If no mark was made, it means no parent was found and there was no replacement.
		// The current dir is added to the results, and checked against future dirs.
		if !hadParent {
			results = append(results, dir)
		}
	}

	return results
}
