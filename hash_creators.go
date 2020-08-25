package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"sort"
)

// AvaiableHashCreators is a map of all HashCreators avaiable in this version of
// the command.
//
// The keys are the string name of the HashCreator, and the value is the HashCreator
// itself.
type AvaiableHashCreators map[string]HashCreator

var avaiableHashCreators = map[string]HashCreator{
	"adler32": func() hash.Hash {
		return hash.Hash(adler32.New())
	},
	"crc32": func() hash.Hash {
		return hash.Hash(crc32.NewIEEE())
	},
	"md5":    md5.New,
	"sha1":   sha1.New,
	"sha256": sha256.New,
	"sha512": sha512.New,
}

// Names returns all HashCreator names in a sorted slice of strings.
func (a AvaiableHashCreators) Names() []string {
	result := make([]string, 0, len(a))
	for name := range a {
		result = append(result, name)
	}

	sort.Strings(result)

	return result
}
