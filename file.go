package main

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"os"
	"sync"
)

// HashCreator is a function that returns a hash.Hash implementation, ready to
// be start hashing data.
type HashCreator func() hash.Hash

// File represents a file to be compared to another.
type File struct {
	hashOnce sync.Once
	path     string
	size     int64
	newHash  HashCreator
	sum      []byte
	sumError error
}

// FileCreator is a function that takes the path and size of a file and returns a
// new, initialized instance of *File.
//
// A more simple function like "NewFile()" is not implemented directly to assure
// cartain consistency issues. For example, all File instances should have the
// same hashing implementation.
type FileCreator func(path string, size int64) *File

// NewFileCreator returns A FileCreator function, ready to start creating *File
// instances.
func NewFileCreator(newHash HashCreator) FileCreator {
	return func(path string, size int64) *File {
		return &File{
			path:    path,
			size:    size,
			newHash: newHash,
		}
	}
}

// doHash calculates the hash of the data of the file.
//
// This is done by using the internal hash.Hash creator function.
//
// No attempt is made to detect if this operation has already been performed for
// this File.
func (f *File) doHash() {
	hasher := f.newHash()

	file, err := os.Open(f.path)
	if err != nil {
		f.sumError = fmt.Errorf("while opening %s for hashing: %v", f.path, err)
		return
	}
	defer file.Close()

	_, err = io.Copy(hasher, file)
	if err != nil {
		f.sumError = fmt.Errorf("while hashing %s: %v", f.path, err)
		return
	}

	f.sum = hasher.Sum(nil)
	f.sumError = nil
}

// Equal compares two instances of File and decides if their contents are equal.
//
// The current algorithm is as follows:
//
// If both Files have different sizes, they're immediately assumed to be different,
// retun False. If they have the same size, they will be hashed.
//
// For both Files, hashes are only calculated when needed, and the result is cached.
func (f *File) Equal(otherF *File) (bool, error) {
	if f.size != otherF.size {
		return false, nil
	}

	// Do the hashing in their own goroutines, but only if they haven't happened yet
	var wg sync.WaitGroup

	f.hashOnce.Do(func() {
		wg.Add(1)

		go func() {
			f.doHash()
			wg.Done()
		}()
	})

	otherF.hashOnce.Do(func() {
		wg.Add(1)

		go func() {
			otherF.doHash()
			wg.Done()
		}()
	})

	wg.Wait()

	if f.sumError != nil {
		return false, f.sumError
	}
	if otherF.sumError != nil {
		return false, otherF.sumError
	}

	return bytes.Equal(f.sum, otherF.sum), nil
}

// Path returns the path of the File.
func (f *File) Path() string {
	return f.path
}

// HashSum returns the hash sum of the contents of the file.
//
// Only calculates hash if not done already. The result, with any error, is cahced.
func (f *File) HashSum() ([]byte, error) {
	f.hashOnce.Do(func() {
		f.doHash()
	})

	return f.sum, f.sumError
}

// Size returns the size of the file.
func (f *File) Size() int64 {
	return f.size
}
