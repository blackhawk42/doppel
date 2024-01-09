package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/blackhawk42/doppel/pkg/collisiondetector"
	"github.com/blackhawk42/doppel/pkg/fileprint"
)

var (
	maxConcurrentFiles int
	bufferSize         int
	uniqueMode         bool
	sortingMode        bool
)

func main() {
	flag.IntVar(&maxConcurrentFiles, "concurrent-files", runtime.NumCPU(), "Maximum number of concurrent files to be hashed. Defaults to the number of detected CPUs.")
	flag.IntVar(&bufferSize, "buffer-size", 4096, "Size of the buffer to use while comparing files.")
	flag.BoolVar(&uniqueMode, "uniques", false, "Unique mode, i. e., report uniques instead of doppelgangers.")
	flag.BoolVar(&sortingMode, "no-sort", true, "Stop sorting of results.")
	flag.Parse()

	collisionDetector := collisiondetector.NewCollisionDetector(int(maxConcurrentFiles))
	filePrintsChan, errorsChan := collisionDetector.Start()

	var wgError sync.WaitGroup
	wgError.Add(1)
	go func() {
		defer wgError.Done()
		for err := range errorsChan {
			log.Println(err)
		}
	}()

	hashingPool := fileprint.NewHashingPool(bufferSize)

	for _, dir := range flag.Args() {
		dir = filepath.Clean(dir)

		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Println(err)
				return nil
			}

			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				log.Println(err)
				return nil
			}

			filePrintsChan <- fileprint.NewFilePrint(path, info.Size(), hashingPool)

			return nil
		})
	}

	close(filePrintsChan)
	wgError.Wait()

	if uniqueMode {
		results := collisionDetector.ReportUniques()

		if sortingMode {
			slices.Sort(results)
		}

		for _, path := range results {
			fmt.Println(path)
		}
	} else {
		results := collisionDetector.ReportCollisions()
		if sortingMode {
			hashes := make([]string, 0, len(results))
			for hash := range results {
				hashes = append(hashes, hash)
			}

			slices.Sort(hashes)

			for _, hash := range hashes {
				fmt.Println(hash)

				paths := results[hash]
				slices.Sort(paths)

				for _, path := range paths {
					fmt.Printf("\t%s\n", path)
				}
			}
		} else {
			for hash, paths := range results {
				fmt.Println(hash)
				for _, path := range paths {
					fmt.Printf("\t%s\n", path)
				}
			}
		}
	}
}
