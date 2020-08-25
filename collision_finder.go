package main

import (
	"fmt"
)

// CollisionFinder constantly receives Files, compares them and stores
// any collision (if a file is equal to another) found.
type CollisionFinder struct {
	sizeCollisions map[int64]*FileList
	hashCollisions map[string]*FileSet
}

// NewCollisionFinder initializes a CollisionFinder.
//
// It receives a the HashCreator function (a function that returns a usable
// implementation). It returns the initialized CollisionFinder and the FileCreator
// function that *must* be used to create all files that will be sent to this
// FileCreator instance.
func NewCollisionFinder(hashCreator HashCreator) (*CollisionFinder, FileCreator) {
	fileCreator := NewFileCreator(hashCreator)

	return &CollisionFinder{
		sizeCollisions: make(map[int64]*FileList),
		hashCollisions: make(map[string]*FileSet),
	}, fileCreator
}

// testCollision tests the given File for collisions with past files, and adds it
// to the pool for future testings.
//
// It receives the File to test and a channel where any errors will be sent.
func (cf *CollisionFinder) testCollision(file *File, errors chan<- error) {
	// Size testing

	// First, we check if this size has been seen before
	fileList, existed := cf.sizeCollisions[file.Size()]
	if !existed {
		fileList = &FileList{}
		cf.sizeCollisions[file.Size()] = fileList

	}

	collision, err := fileList.CollisionAdd(file)
	if err != nil {
		go func() {
			errors <- fmt.Errorf("while checking for size collisions: %v", err)
		}()

		return
	}

	// If no collision was found during insertion, everything cool...
	if collision == nil {
		return
	}

	// Sum testing

	// ... but if it was, we have to add it to the set, based on its sum
	stringSum := fmt.Sprintf("%x", collision.Sum)
	collisionSet, existed := cf.hashCollisions[stringSum]
	if !existed {
		collisionSet = NewFileSet()
		cf.hashCollisions[stringSum] = collisionSet
	}

	collisionSet.Add(collision.FirstFile, collision.SecondFile)
}

// report generates a report (as specified in the documentation for Run) and sends
// it through the given channel.
func (cf *CollisionFinder) report(reportChan chan<- map[string][]string) {
	report := make(map[string][]string)

	for hash, files := range cf.hashCollisions {
		paths := make([]string, 0, files.Len())
		for _, file := range files.Slice() {
			paths = append(paths, file.Path())
		}

		report[hash] = paths
	}

	reportChan <- report
}

// Run sets up a new goroutine where the CollisionFinder will continously compare
// files.
//
// It returns three channels.
//
// The first channel should only be used to send Files to check for collisions.
// If you send, and the sending was sucessfull, the request is guaranteed to be
// processed before another request is accepted, or a report can be made.
//
// The second channel has a double purpose. First, you send to it to signal that you
// want to receive a report. This signal will be discarded, so you should probably
// just send nil. All activity will cease, a report will be generated, and sent through
// this same channel. Activity will only proceed when the report is succesfully
// received.
//
// The third channel is simply where any errors at any part of the operation will
// be sent through. Errors will wait at the other side of the channel until they
// are received, and no order is assured. Error sending never blocks operations,
// so you could in theory ignore this channel altogether. But we wouldn't do that,
// would we?.
//
// The format of the report is as follows:
//
// Each key in the map is a string with the hex representation of the hash, and
// the values are the Paths of the Files that were found to collide with that hash,
// in no particular order.
func (cf *CollisionFinder) Run() (chan<- *File, chan map[string][]string, <-chan error) {
	requests := make(chan *File)
	report := make(chan map[string][]string)
	errors := make(chan error)

	go func() {
		for {
			select {
			case file := <-requests:
				cf.testCollision(file, errors)
			case <-report:
				cf.report(report)
			}
		}
	}()

	return requests, report, errors
}
