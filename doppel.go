package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

func main() {

	sizeMap := NewSizeMap(NewFileCollisions())
	filesChan := make(chan *File)
	collisionsChan := make(chan map[string][]string)

	go getCollisions(sizeMap, filesChan, collisionsChan)

	var wg sync.WaitGroup
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			wg.Add(1)
			go func() {
				filesChan <- NewFile(path, info.Size())
				wg.Done()
			}()
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	wg.Wait()
	close(filesChan)

	for hash, files := range <-collisionsChan {
		fmt.Printf("%s\n", hash)
		sort.Strings(files)
		for _, file := range files {
			fmt.Printf("\t%s\n", file)
		}
		fmt.Printf("\n")
	}

}

func getCollisions(sizeMap *SizeMap, filesChan <-chan *File, collisionsChan chan<- map[string][]string) {
	for file := range filesChan {
		sizeMap.Add(file)
	}

	collisionsChan <- sizeMap.Collisions()
}
