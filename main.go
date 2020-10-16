package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Flag setting

// Default flags
const (
	DefaultHashCreatorName = "sha1"
	DefaultSerialWalkMode  = false
)

// Flag variables
var (
	HashCreatorName = flag.String("hash", DefaultHashCreatorName, "`name` of the hash function to use")
	SerialWalkMode  = flag.Bool("serial-walk", DefaultSerialWalkMode, "walk root folders one by one, instead of the default of starting a porces for each root directory")
)

func main() {
	// Flags and sanity checks

	flag.Usage = func() {
		av := AvaiableHashCreators(avaiableHashCreators)

		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [FLAGS] ROOT_DIR [ROOT_DIR...]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(flag.CommandLine.Output(), "Recursively search multiple \"root\" directories for duplicate files.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Avaiable hash functions: %s.\n", strings.Join(av.Names(), ", "))
	}

	flag.Parse()

	if _, exists := avaiableHashCreators[*HashCreatorName]; !exists {
		fmt.Fprintf(os.Stderr, "error: %s is not a valid hash function name\n", *HashCreatorName)
		flag.Usage()
		os.Exit(2)
	}

	// If given with no args, take it as a valid way to ask for help. The risk of
	// running it accidentally in a big directory is too high, so all runs should
	// be explicit.
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	// Sanitize directories, delete repeats
	rootDirs := make([]string, len(flag.Args()))

	for i, dir := range flag.Args() {
		sanitized, err := sanitizeDir(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			flag.Usage()
			os.Exit(2)
		}

		rootDirs[i] = sanitized
	}

	rootDirs = normalizeDirs(rootDirs)

	// Main logic

	cf, newFile := NewCollisionFinder(avaiableHashCreators[*HashCreatorName])
	requests, reportChan, cfErrors := cf.Run()

	// Log all found errors as they come
	go func() {
		for err := range cfErrors {
			log.Print(err)
		}
	}()

	wg := new(sync.WaitGroup)
	for _, dir := range rootDirs {
		// In serial mode, we just call the worker for each passed root, same parameters
		if *SerialWalkMode {
			err := walkWorker(dir, newFile, requests)
			if err != nil {
				log.Print(err)
			}

			// Else, we start a new goroutine for each directory
		} else {
			wg.Add(1)

			go func(dir string) {
				defer wg.Done()

				err := walkWorker(dir, newFile, requests)
				if err != nil {
					log.Print(err)
				}

			}(dir)
		}
	}
	wg.Wait()

	reportChan <- nil
	report := <-reportChan

	// Sort results
	resultSums := make([]string, 0, len(report))
	for sum := range report {
		resultSums = append(resultSums, sum)
	}
	sort.Strings(resultSums)

	for _, sum := range resultSums {
		fmt.Printf("%s\n", sum)
		files := report[sum]

		sort.Strings(files)
		for _, f := range files {
			fmt.Printf("\t%s\n", f)
		}
	}
}

func walkWorker(rootDir string, newFile FileCreator, requests chan<- *File) error {
	// Main logic

	wg := new(sync.WaitGroup)
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log error an try to carry on with the rest of the files
			log.Print(err)
			return nil
		}

		if !info.IsDir() {
			wg.Add(1)
			go func() {
				defer wg.Done()

				requests <- newFile(path, info.Size())
			}()
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("while walking through %s: %v", rootDir, err)
	}

	wg.Wait()

	return nil
}
