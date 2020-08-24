package main

import (
	"fmt"
	"sync"
)

// CollisionFinder constantly receives Files, compares them and stores
// any collision (if a file is equal to another) found.
type CollisionFinder struct {
	sizeCollisionsMutex sync.Mutex
	sizeCollisions      map[int64]*FileList
	hashCollisionsMutex sync.Mutex
	hashCollisions      map[string]*FileSet

	runGroup             sync.WaitGroup
	comparisonsSemaphore *Semaphore
}

// NewCollisionFinder initializes a CollisionFinder.
//
// maxConcurrentComparisns are the maximum number of comparisons that may occur
// concurrently. Will panic if <= 0.
func NewCollisionFinder(maxConcurrentComparisons int) *CollisionFinder {
	if maxConcurrentComparisons <= 0 {
		panic("maxConcurrentComparisons must be > 0")
	}

	return &CollisionFinder{
		sizeCollisions:       make(map[int64]*FileList),
		hashCollisions:       make(map[string]*FileSet),
		comparisonsSemaphore: NewSemaphore(maxConcurrentComparisons),
	}
}

// Run sets up a new goroutine where the CollisionFinder will continously compare
// compare files.
//
// It returns a channel, where the *Files will be received, and
// returns a channel, where any found error will be sent.
//
// Close the File channel when done sending comparison requests.
func (cf *CollisionFinder) Run() (chan<- *File, <-chan error) {
	requests := make(chan *File)
	errors := make(chan error)

	go func() {
		for file := range requests {
			cf.runGroup.Add(1)
			go func(file *File) {
				defer cf.runGroup.Done()

				// Size testing

				// First, we check if this size has been seen before
				cf.sizeCollisionsMutex.Lock() // Has to be unlocked without defer. Careful!
				fileList, existed := cf.sizeCollisions[file.Size()]
				if !existed {
					fileList = &FileList{}
					cf.sizeCollisions[file.Size()] = fileList

				}

				cf.comparisonsSemaphore.Acquire() // Another with no defer. Careful!
				collision, err := fileList.CollisionAdd(file)
				cf.comparisonsSemaphore.Done()
				if err != nil {
					go func() {
						errors <- fmt.Errorf("while checking for size collisions: %v", err)
					}()

					cf.sizeCollisionsMutex.Unlock()
					return
				}
				cf.sizeCollisionsMutex.Unlock()

				// If no collision was found during insertion, everything cool...
				if collision == nil {
					return
				}

				// ... but if it was, we have to add it to the set, based on its sum
				stringSum := fmt.Sprintf("%x", collision.Sum)
				cf.hashCollisionsMutex.Lock() // Again, careful!
				collisionSet, existed := cf.hashCollisions[stringSum]
				if !existed {
					collisionSet = NewFileSet()
					cf.hashCollisions[stringSum] = collisionSet
				}

				collisionSet.Add(collision.FirstFile, collision.SecondFile)
				cf.hashCollisionsMutex.Unlock()

			}(file)
		}
	}()

	return requests, errors
}

// Result returns a channel with a map of all collisions found.
//
// This method waits for all File channels created with Run to be closed. Then,
// it will generate a map with all collisions found and return it via the retuned
// channel. Because the map won't actually be sent until all current jobs are
// finished, onecan use this channel both to receive the final report and the
// sending itself as a signal to procceed.
//
// Each key in the map is a string with the hex representation of the hash, and
// the values are the Paths of the Files that were found to collide with that hash,
// in no particular order.
func (cf *CollisionFinder) Result() <-chan map[string][]string {
	reportChan := make(chan map[string][]string)

	go func() {
		cf.runGroup.Wait()
		report := make(map[string][]string)

		cf.hashCollisionsMutex.Lock()
		defer cf.hashCollisionsMutex.Unlock()

		for hash, files := range cf.hashCollisions {
			paths := make([]string, 0, files.Len())
			for _, file := range files.Slice() {
				paths = append(paths, file.Path())
			}

			report[hash] = paths
		}

		reportChan <- report
	}()

	return reportChan
}
