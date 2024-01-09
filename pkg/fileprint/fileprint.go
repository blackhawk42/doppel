package fileprint

import (
	"hash"
	"io"
	"os"
	"sync"

	"lukechampine.com/blake3"
)

// HashingPool implements a thread-safe pools of useful resources when doing hashing.
type HashingPool struct {
	bufferPool sync.Pool
	hasherPool sync.Pool
}

// newBuffer creates a new buffer of the specified size and returns a pointer to it (*[]byte).
//
// For use as the New function in an appropiate sync.Pool
func newBuffer(size int) any {
	buffer := make([]byte, size)
	return &buffer
}

// newBuffer creates a new hash.Hash.
//
// For use as the New function in an appropiate sync.Pool
func newHasher() any {
	return blake3.New(32, nil)
}

// NewHashingPool creates a new HashingPool.
//
// bufferSize specifies the size of the buffers it will return.
func NewHashingPool(bufferSize int) *HashingPool {
	return &HashingPool{
		bufferPool: sync.Pool{
			New: func() any { return newBuffer(bufferSize) },
		},
		hasherPool: sync.Pool{
			New: newHasher,
		},
	}
}

// GetBuffer returns a buffer from the pool, creating it if there isn't any.
func (hp *HashingPool) GetBuffer() *[]byte {
	return hp.bufferPool.Get().(*[]byte)
}

// GetHasher returns a buffer from the pool, creating it if there isn't any.
func (hp *HashingPool) GetHasher() hash.Hash {
	return hp.hasherPool.Get().(hash.Hash)
}

// PutBuffer puts a buffer back int the pool.
//
// Should be called when done with the buffer.
func (hp *HashingPool) PutBuffer(buffer *[]byte) {
	hp.bufferPool.Put(buffer)
}

// PutBuffer puts a hasher back int the pool, and resets it.
//
// Should be called when done with the hasher.
func (hp *HashingPool) PutHasher(hasher hash.Hash) {
	hasher.Reset()
	hp.hasherPool.Put(hasher)
}

// FilePrint represents a sort of "fingerprint" for a file in memory.
//
// Mainly, it stores the path of the file, it's size, and a hash of the file.
// That last one is computed lazily (only when requested, and only once), in a thread-safe way.
type FilePrint struct {
	path         string
	size         int64
	hashingPool  *HashingPool
	hashOnce     sync.Once
	hash         []byte
	hashingError error
}

// NewFilePrint creates a new FilePrint.
func NewFilePrint(path string, size int64, pool *HashingPool) *FilePrint {
	return &FilePrint{
		path:        path,
		size:        size,
		hashingPool: pool,
	}
}

// Path returns the path of the file.
func (fp *FilePrint) Path() string {
	return fp.path
}

// Size returns the size of the file.
func (fp *FilePrint) Size() int64 {
	return fp.size
}

// Hash returns the hash of the file, computing it if needed.
//
// No new threads are started, but the computation itself is thread-safe,
// so can be done inside a new one, to the discretion of the caller.
//
// Current hash is BLAKE3 with length of 256 bits.
func (fp *FilePrint) Hash() ([]byte, error) {
	fp.hashOnce.Do(func() {
		f, err := os.Open(fp.path)
		if err != nil {
			fp.hashingError = err
			return
		}
		defer f.Close()

		buffer := fp.hashingPool.GetBuffer()
		defer fp.hashingPool.PutBuffer(buffer)

		hasher := fp.hashingPool.GetHasher()
		defer fp.hashingPool.PutHasher(hasher)

		_, err = io.CopyBuffer(hasher, f, *buffer)
		if err != nil {
			fp.hashingError = err
			return
		}

		fp.hash = hasher.Sum(nil)
	})

	return fp.hash, fp.hashingError
}

// FilePrintSlice is a simple slice of pointers to FilePrints.
//
// Mostly to implement the sort.Interface, with the path as the ID.
type FilePrintSlice []*FilePrint

func (fps FilePrintSlice) Len() int {
	return len(fps)
}

func (fps FilePrintSlice) Less(i, j int) bool {
	return fps[i].path < fps[j].path
}

func (fps FilePrintSlice) Swap(i, j int) {
	fps[i], fps[j] = fps[j], fps[i]
}
