package main

// SizeMap represents a relationship between a size and all files detected
// to be of that size.
type SizeMap struct {
	m          map[int64][]*File
	collisions *FileCollisions
}

// NewSizeMap Creates and initializes a new SizeMap.
//
// collisions must be an initialized *FileCollisions.
func NewSizeMap(collisions *FileCollisions) *SizeMap {
	return &SizeMap{
		m:          make(map[int64][]*File),
		collisions: collisions,
	}
}

// Add associates a file to it's size.
//
// If the same file is added twice, it will be
// considered a collision.
func (sm *SizeMap) Add(f *File) {
	fSize := f.Size()

	if otherFiles, exists := sm.m[fSize]; exists {
		for _, otherFile := range otherFiles {
			if otherFile.Equals(f) {
				sm.collisions.Add(f)
				sm.collisions.Add(otherFile)
			}
		}

		sm.m[fSize] = append(otherFiles, f)
	} else {
		fileSlice := make([]*File, 0)
		fileSlice = append(fileSlice, f)
		sm.m[fSize] = fileSlice
	}
}

// Collisions returns all found collisions until now.
func (sm *SizeMap) Collisions() map[string][]string {
	return sm.collisions.Collisions()
}
