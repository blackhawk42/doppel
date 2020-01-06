package main

import (
	"fmt"
)

// FileCollisions defines a list of key-value pairs, where the key is the hash
// sum of a file and the value is a StringSet containing the paths of every file
// found to be doppelgängers of each other.
type FileCollisions struct {
	m map[string]*StringSet
}

// NewFileCollisions creates and initializes a new FileCollisions.
func NewFileCollisions() *FileCollisions {
	fc := &FileCollisions{
		m: make(map[string]*StringSet),
	}

	return fc
}

// Add inserts a File to the list of files that have been equal to it.
//
// Safe to call multiple times with the same file.
func (fc *FileCollisions) Add(f *File) {
	key := fmt.Sprintf("%X", f.HashSum())

	if _, exists := fc.m[key]; !exists {
		fc.m[key] = NewStringSet()
	}

	fc.m[key].Add(f.Path())
}

// Collisions returns a map associating a hash value to all paths found to have
// collided with said hash sum.
func (fc *FileCollisions) Collisions() map[string][]string {
	result := make(map[string][]string)

	for k := range fc.m {
		result[k] = fc.m[k].Members()
	}

	return result
}
