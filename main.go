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
)

// Flag variables
var (
	HashCreatorName = flag.String("hash", DefaultHashCreatorName, "`name` of the hash function to use")
)

func main() {
	// Flags and sanity checks

	flag.Usage = func() {
		av := AvaiableHashCreators(avaiableHashCreators)

		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [FLAGS] [ROOT_DIR]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(flag.CommandLine.Output(), "Recursively search a directory and subdirectories for duplicate files.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Avaiable hash functions: %s.\n\n", strings.Join(av.Names(), ", "))
		fmt.Fprintf(flag.CommandLine.Output(), `The general algorithm is as follows: if two files have different sizes,
they're inmediately considered different. If they have the same size,
they're hashed (only calculated once per file, when needed) and their
sums are compared for a final veredict.
		
`)
		fmt.Fprintf(flag.CommandLine.Output(), `The final report has the following format:

HEX_HASH_SUM
	CLONED_FILE_1
	CLONED_FILE_2
	...

And so on, for every group of files that were found to be clones. Results
are sorted so, in theory, multiple runs in the same directory should
produce the same output, if the files don't change.

`)
		fmt.Fprintf(flag.CommandLine.Output(), `Remember not to take the results as gospel! No byte-for-byte comparison is
attempted to absolutely confirm two collisions are equal. False positves
with a good hash are unlikely, but not impossible. If the risk is not
acceptable, don't do something like passing the output directly to a
deleting script without byte-for-byte or manual double checking.

`)
		fmt.Fprintf(flag.CommandLine.Output(), `Obviously, the more files are compared, the more memory is required for
the internal structure, even if only the strictly necessary is saved
(names, sizes, and cached hashes). Also, the hash function is necessarily
a compromise between hashing speed, size when cached, and probability
of collision. The default (SHA-1) was chosen for being relatively
speedy in modern machines and having an almost negligible chance of
collision in practice (this application is *not* meant for security
hardiness). Of course, some other algorithms are provided, but keep in
mind: the bottleneck will probably be just the tree traversing and all
those stat calls. In any case, hashing should only come into play in the
case of many files with the exact same size, which should be, hopefully,
uncommon in most practical cases.

`)
	}

	flag.Parse()

	if _, exists := avaiableHashCreators[*HashCreatorName]; !exists {
		fmt.Fprintf(os.Stderr, "error: %s is not a valid hash function name\n", *HashCreatorName)
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
		fmt.Fprintf(os.Stderr, "error: given root is not a directory\n")
		flag.Usage()
		os.Exit(2)
	}

	// Main logic

	cf, newFile := NewCollisionFinder(avaiableHashCreators[*HashCreatorName])
	requests, reportChan, errors := cf.Run()

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
				defer wg.Done()

				requests <- newFile(path, info.Size())
			}()
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
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
