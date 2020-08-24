package main

import (
	"fmt"
)

// FileCollision represents a collision between two files.
type FileCollision struct {
	FirstFile  *File
	SecondFile *File
	Sum        []byte
}

// FileListNode is a node inside a FileList.
type FileListNode struct {
	File *File
	Next *FileListNode
}

// FileList is a simply-linked list of Files.
//
// It can add a file in constant time or in linear time, testing if the File
// is equal to any of the already existing Files in the list.
//
// An empty FileList is a list ready for use.
type FileList struct {
	len   int
	first *FileListNode
	last  *FileListNode
}

// Len returns the current length of the FileList.
//
// Constant time.
func (fList *FileList) Len() int {
	return fList.len
}

// Add inserts a new File into the list in constant time, with no tests done.
func (fList *FileList) Add(file *File) {
	fList.len++

	newNode := &FileListNode{
		File: file,
	}

	if fList.first == nil {
		fList.first = newNode
		fList.last = newNode
	} else {
		fList.last.Next = newNode
		fList.last = newNode
	}
}

// CollisionAdd adds a File to the list, detecting if it collides with any of the
// already existing Files.
//
// Basically, it takes the receiving File and calls its Equal method on every File
// already existing on the list. If a collision is found, A new FileCollision is
// created and returned, with a nil error. Otherwise, nil is returned for both values. If an error
// happens, the FileCollision will be nil, and no change will be made to the list.
//
// Insertion time is linear to the current length of the list until the first collision.
// The first collision found is returned. Also, this is not a set: the given File
// will always be added, even if a collision is found.
func (fList *FileList) CollisionAdd(file *File) (*FileCollision, error) {
	var result *FileCollision

	// Search for a collision
	for current := fList.first; current != nil; current = current.Next {
		collision, err := file.Equal(current.File)
		if err != nil {
			return nil, fmt.Errorf("while inserting to list: %v", err)
		}

		if collision {
			hashSum, err := file.HashSum()
			if err != nil {
				return nil, fmt.Errorf("while inserting to list and getting hash: %v", err)
			}

			result = &FileCollision{
				FirstFile:  file,
				SecondFile: current.File,
				Sum:        hashSum, // Should be the same for both.
			}

			break
		}
	}

	fList.Add(file)

	return result, nil
}

// Slice returns the linked list as a slice.
//
// Time linear to the number of Files in the list.
func (fList *FileList) Slice() []*File {
	slice := make([]*File, 0, fList.len)

	for current := fList.first; current != nil; current = current.Next {
		slice = append(slice, current.File)
	}

	return slice
}
