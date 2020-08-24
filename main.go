package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Default flags
const (
	DefaultConcurrentComparisons = 10
)

// Flag setting
var (
	concurrentComparisons = flag.Int("workers", DefaultConcurrentComparisons, "`number` (> 0) of possible concurrent comparisons; not exactly a \"worker\" in the usual sense, but easier to rememeber")
)

func main() {
	// Flags and sanity checks

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [FLAGS] [ROOT_DIR]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(flag.CommandLine.Output(), "Recursively search a directory for duplicate files.\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	if *concurrentComparisons <= 0 {
		fmt.Fprintf(os.Stderr, "workers must be > 0\n")
		flag.Usage()
		os.Exit(2)
	}

	// If given with no args, attempt to use in current working directory.
	var rootDir string
	var err error
	if len(flag.Args()) == 0 {
		rootDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while getting current working directory: %v\n", err)
			flag.Usage()
			os.Exit(2)
		}
	} else {
		rootDir, err = filepath.Abs(flag.Args()[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while getting absolute path of given directory %s: %v\n", flag.Args()[0], err)
			flag.Usage()
			os.Exit(2)
		}
	}

	rootDirInfo, err := os.Stat(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while getting infromation of given directory %s: %v\n", rootDir, err)
		flag.Usage()
		os.Exit(2)
	}

	if !rootDirInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "given root is not a directory\n")
		flag.Usage()
		os.Exit(2)
	}

	// Main logic

	cf := NewCollisionFinder(*concurrentComparisons)

	newFile := NewFileCreator(sha1.New)

	requests, errors := cf.Run()

	// Log all found errors as they come
	go func() {
		for err := range errors {
			log.Print(err)
		}
	}()

	var wg sync.WaitGroup
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log error an try to carry on with the rest of the files
			log.Print(err)
			return nil
		}

		if !info.IsDir() {
			wg.Add(1)
			go func() {
				requests <- newFile(path, info.Size())
				wg.Done()
			}()
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()
	close(requests)

	result := <-cf.Result()

	// Sort results
	resultSums := make([]string, 0, len(result))
	for sum := range result {
		resultSums = append(resultSums, sum)
	}
	sort.Strings(resultSums)

	for _, sum := range resultSums {
		fmt.Printf("%s\n", sum)
		files := result[sum]

		sort.Strings(files)
		for _, f := range files {
			fmt.Printf("\t%s\n", f)
		}
	}

}
