package collisiondetector

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/blackhawk42/doppel/pkg/fileprint"
	"github.com/blackhawk42/doppel/pkg/semaphore"
	mapset "github.com/deckarep/golang-set/v2"
)

// collisionOrder is a utility struct to easily pass these two pointers through
// a channel.
//
// Probably not really necessary with the current design, but it saves a map lookup
// and string conversion, so why not?
type collisionOrder struct {
	fp1 *fileprint.FilePrint
	fp2 *fileprint.FilePrint
}

// CollisionDetector takes a stream of FilePrints and detects collisions, i. e.,
// files with the same contetns but different names.
//
// After creation with NewCollisionDetector, Start should be called to use it.
// See Start documentation for important details.
type CollisionDetector struct {
	concurrency        int
	concurrentFilesSem *semaphore.Semaphore
	sizeFilePrints     map[int64]fileprint.FilePrintSlice
	collisions         map[string]mapset.Set[string]
	inputs             chan *fileprint.FilePrint
	toAddCollision     chan collisionOrder
	errors             chan error
}

// NewCollisionDetector creates a new CollisionDetector.
func NewCollisionDetector(concurrentFiles int) *CollisionDetector {
	return &CollisionDetector{
		concurrency:        concurrentFiles,
		concurrentFilesSem: semaphore.NewSemaphore(concurrentFiles),
		sizeFilePrints:     make(map[int64]fileprint.FilePrintSlice),
		collisions:         make(map[string]mapset.Set[string]),
	}
}

// comparisonProcess is the first step in the internal pipeline.
//
// CollisionDetector works on a sort of pipeline. First, FilePrints enter here,
// where they are put in a map relating to size. If files with the same size
// exist, a subprocess is started to properly compare them. If they are indeed
// doppelgangers, yet another subprocess send them to the next step in the pipeline.
//
// Each time an error can occur, we create a subprocess to asyncronously
// report it to the error stream and keep going.
//
// When the input channel is closed, we wait for all subprocesses to end. Onces
// the pipes are drained, as it were, we close the channel to the next step.
func (cd *CollisionDetector) comparisonProcess() {
	var wg sync.WaitGroup
	for fp := range cd.inputs {
		fps, ok := cd.sizeFilePrints[fp.Size()]
		if !ok {
			fps = make(fileprint.FilePrintSlice, 0)
		}

		for _, otherFp := range fps {
			wg.Add(1)
			go func(fp1 *fileprint.FilePrint, fp2 *fileprint.FilePrint) {
				defer wg.Done()

				doppel, err := cd.areDoppel(fp1, fp2)
				if err != nil {
					wg.Add(1)
					go func() {
						defer wg.Done()

						cd.errors <- err
					}()

					return
				}

				if doppel {
					wg.Add(1)
					go func() {
						defer wg.Done()

						cd.toAddCollision <- collisionOrder{fp1: fp1, fp2: fp2}
					}()
				}
				// The FP already registered is more likely to have been activated,
				// so it's better for it to go first, for when it's hash is innevitably
				// used if found to store the collision (if found to collide at all).
			}(otherFp, fp)
		}

		cd.sizeFilePrints[fp.Size()] = append(fps, fp)
	}

	wg.Wait()
	close(cd.toAddCollision)
}

// addCollisionProcess is the second step in the internal pipeline.
//
// It just receives detected collisions and puts them in a set (so repeated files
// don't matter) mapped to their hash. Again, errors are passed through an asyncronous
// subprocess.
//
// When the toAddCollision channel closes, it means the first step has closed
// and all subprocesses have ended, so we just close the errors channel (which can
// be seen as the third step of the pipeline).
func (cd *CollisionDetector) addCollisionProcess() {
	var wg sync.WaitGroup
	for order := range cd.toAddCollision {
		hash, err := order.fp1.Hash()
		if err != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cd.errors <- err
			}()

			continue
		}

		hashString := hex.EncodeToString(hash)
		cols, ok := cd.collisions[hashString]
		if !ok {
			cols = mapset.NewSet[string]()
			cd.collisions[hashString] = cols
		}

		cols.Add(order.fp1.Path())
		cols.Add(order.fp2.Path())
	}
	wg.Wait()
	close(cd.errors)
}

// Start initializes the CollisionDetector and makes it ready for use.
//
// It returns two channels. The first one is for FilePrints to be sent to. The
// seconds communicates errors at any step of the process, if any.
//
// To shutdown the CollisionDetector, close the first channel and "drain" the
// errors channel (receive any pending errors and wait for it to close).
func (cd *CollisionDetector) Start() (chan<- *fileprint.FilePrint, <-chan error) {
	cd.inputs = make(chan *fileprint.FilePrint, cd.concurrency)
	cd.toAddCollision = make(chan collisionOrder, cd.concurrency)
	cd.errors = make(chan error)

	go cd.comparisonProcess()
	go cd.addCollisionProcess()

	return cd.inputs, cd.errors
}

// doppelResult is an utility struct to receive a hashing result through a channel.
type doppelResult struct {
	path string
	hash []byte
	err  error
}

// areDoppel compares to FilePrints and decides if they are doppelgangers.
//
// It does it first based on size, then based on path, and lastly hash. It tries to hash them
// concurrently, based on the CollisionDetector semaphore to set a limit.
func (cd *CollisionDetector) areDoppel(fp1, fp2 *fileprint.FilePrint) (bool, error) {
	if fp1.Size() != fp2.Size() {
		return false, nil
	}

	if fp1.Path() == fp2.Path() {
		return true, nil
	}

	results := make(chan doppelResult)

	go func() {
		cd.concurrentFilesSem.Acquire()
		defer cd.concurrentFilesSem.Release()

		hash, err := fp1.Hash()
		results <- doppelResult{path: fp1.Path(), hash: hash, err: err}
	}()

	go func() {
		cd.concurrentFilesSem.Acquire()
		defer cd.concurrentFilesSem.Release()

		hash, err := fp2.Hash()
		results <- doppelResult{path: fp2.Path(), hash: hash, err: err}
	}()

	result1 := <-results
	if result1.err != nil {
		return false, fmt.Errorf("while hashing FilePrint for %s: %w", result1.path, result1.err)
	}

	result2 := <-results
	if result2.err != nil {
		return false, fmt.Errorf("while hashing FilePrint for %s: %w", result2.path, result2.err)
	}

	return bytes.Equal(result1.hash, result2.hash), nil
}

// ReportCollisions creates a "report" mapping each hash with all the files
// (by path) detected to be doppelgangers.
func (cd *CollisionDetector) ReportCollisions() map[string][]string {
	result := make(map[string][]string, len(cd.collisions))

	for hash, set := range cd.collisions {
		result[hash] = set.ToSlice()
	}

	return result
}

// ReportUniques creates a slice of paths deemed to be unique (i. e., without
// doppelgangers).
func (cd *CollisionDetector) ReportUniques() []string {
	result := make([]string, 0)

	var hasCollision bool
	for _, fps := range cd.sizeFilePrints {
		// For every found file, no matter it's size...
		for _, fp := range fps {
			hasCollision = false
			for _, collisionedFps := range cd.collisions {
				// See if it has been registered in any of the collisions sets
				if collisionedFps.ContainsOne(fp.Path()) {
					hasCollision = true
					break
				}
			}

			// Only if not found, do we count it
			if !hasCollision {
				result = append(result, fp.Path())
			}
		}
	}

	return result
}
