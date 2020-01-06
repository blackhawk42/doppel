package main

import (
	"bytes"
	"crypto/md5"
	"io"
	"os"
	"sync"
)

// File represents a file found in the filesystem, with anything deemed necessary
// to check if it is equal to another.
//
// This struct contains the hash of the file. It only calculated once, the first
// time it's needed; a sync.Once instance used for this purpose.
type File struct {
	path string
	size int64

	doHashing sync.Once
	hashSum   []byte
}

func (f *File) calculateHashSum() error {
	fOpened, err := os.Open(f.path)
	if err != nil {
		return err
	}
	defer fOpened.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, fOpened); err != nil {
		return err
	}

	f.hashSum = hasher.Sum(nil)

	return nil
}

// NewFile is a helper function for creating a new File.
func NewFile(path string, size int64) *File {
	return &File{
		path: path,
		size: size,
	}
}

// Equals determines if this File is equal to another.
//
// "Equal" here relates to their contents. Two files on different filesystem
// locations are equal if and only if their contents are identical.
//
// The method tries a series of tests in order of simplicity (and, it's hoped, speed)
// to determine equalness, and the method returns as soon as an answer is found.
// These tests, in order, are:
// 1. If the files have a different size, they're assumed to be different; return false.
// Else, they may coincidentally have the same size; continue testing.
// 2. If the files have the same location in the filesystem, or path, they're
// assumed to be equal; return true. Else, we have a file that has the same size
// on a different location; continue testing.
// 3. Calculate this and the other's MD5 hash sum. If both sums are equal, the files
// are assumed to be equal, return true. Else, they're assumed to be different, return
// false.
//
// It is safe to call this method more than once. It is also thread-safe.
func (f *File) Equals(otherF *File) bool {
	if f.size != otherF.size {
		return false
	}

	if f.path == otherF.path {
		return true
	}

	// Do the hash calculation of the two files concurrently
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		f.doHashing.Do(func() {
			f.calculateHashSum()
		})
		wg.Done()
	}()

	go func() {
		otherF.doHashing.Do(func() {
			otherF.calculateHashSum()
		})
		wg.Done()
	}()

	wg.Wait()

	return bytes.Equal(f.hashSum, otherF.hashSum)
}

// Path returns the path location of this file.
func (f *File) Path() string {
	return f.path
}

// Size returns the size of the associated file
func (f *File) Size() int64 {
	return f.size
}

// HashSum returns the hash sum of the file.
//
// It is safe to call this method more than once. It is also thread-safe.
func (f *File) HashSum() []byte {
	f.doHashing.Do(func() {
		f.calculateHashSum()
	})
	return f.hashSum
}
