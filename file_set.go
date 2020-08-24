package main

// FileSet implements a set of Files by Path.
//
// Files with the same Path will only be included only once. Multiple Files
// may have the same HashSum.
type FileSet struct {
	set map[string]*File
}

// NewFileSet returns an initialized FileSet.
func NewFileSet() *FileSet {
	return &FileSet{
		set: make(map[string]*File),
	}
}

// Add a File to the set.
//
// Repeated elements should only be added once to the overall set. If for some
// reason two Files have the same Path, the last passed element will be kept.
func (fSet *FileSet) Add(files ...*File) {
	for _, file := range files {
		fSet.set[file.Path()] = file
	}
}

// Len returns the amount of current Files in the set.
func (fSet *FileSet) Len() int {
	return len(fSet.set)
}

// Slice returns a slice of all current Files in the set.
//
// No order on returned elements is guaranteed.
func (fSet *FileSet) Slice() []*File {
	slice := make([]*File, 0, len(fSet.set))
	for _, file := range fSet.set {
		slice = append(slice, file)
	}

	return slice
}
