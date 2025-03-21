package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/blackhawk42/doppel/pkg/collisiondetector"
	"github.com/blackhawk42/doppel/pkg/fileprint"
)

type CLI struct {
	MaxConcurrentFiles int      `default:"${DEFAULT_CONCURRENT_FILES}" short:"c" help:"Maximum number of concurrent files to be hashed. Defaults to the number of detected CPUs."`
	BufferSize         int      `default:"${DEFAULT_BUFFER_SIZE}" short:"b" help:"Size of the buffer to use while comparing files."`
	UniquesMode        bool     `default:"false" short:"u" help:"Unique mode, i. e., report uniques instead of doppelgangers."`
	OutputFile         string   `optional:"" short:"o" type:"path" help:"Output file. Stdout by default."`
	Paths              []string `arg:"" type:"existingdir" help:"Paths to directories from where to run the search."`
}

func main() {
	cli := CLI{}
	kongCtx := kong.Parse(
		&cli,
		kong.Description("An utility to find files with different names but same contents"),
		kong.Vars{
			"DEFAULT_CONCURRENT_FILES": fmt.Sprint(runtime.NumCPU()),
			"DEFAULT_BUFFER_SIZE":      fmt.Sprint(4096),
		},
	)

	var ouptut = os.Stdout
	if cli.OutputFile != "" {
		f, err := os.Create(cli.OutputFile)
		kongCtx.FatalIfErrorf(err, fmt.Sprintf("while opening output file: %v", err))

		ouptut = f
		defer f.Close()
	}

	collisionDetector := collisiondetector.NewCollisionDetector(cli.MaxConcurrentFiles)
	filePrintsChan, errorsChan := collisionDetector.Start()

	var wgError sync.WaitGroup
	wgError.Add(1)
	go func() {
		defer wgError.Done()
		for err := range errorsChan {
			kongCtx.Errorf("error: %v", err)
		}
	}()

	hashingPool := fileprint.NewHashingPool(cli.BufferSize)

	for _, dir := range cli.Paths {
		dir = filepath.Clean(dir)

		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				kongCtx.Errorf("error while walking %s: %v", dir, err)
				return nil
			}

			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				kongCtx.Errorf("error while walking %s: %v", dir, err)
				return nil
			}

			filePrintsChan <- fileprint.NewFilePrint(path, info.Size(), hashingPool)

			return nil
		})
	}

	close(filePrintsChan)
	wgError.Wait()

	if cli.UniquesMode {
		results := collisionDetector.ReportUniques()

		slices.Sort(results)

		for _, path := range results {
			fmt.Fprintln(ouptut, path)
		}
	} else {
		results := collisionDetector.ReportCollisions()

		// Sorting
		hashes := make([]string, 0, len(results))
		for hash := range results {
			hashes = append(hashes, hash)
		}

		slices.Sort(hashes)

		for _, hash := range hashes {
			fmt.Fprintln(ouptut, hash)

			paths := results[hash]
			slices.Sort(paths)

			for _, path := range paths {
				fmt.Fprintf(ouptut, "\t%s\n", path)
			}
		}
	}
}
