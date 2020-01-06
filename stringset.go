package main

var setMember struct{}

// StringSet is a simple set of strings.
type StringSet struct {
	m map[string]interface{}
}

// NewStringSet creates and initializes a StringSet.
func NewStringSet() *StringSet {
	set := &StringSet{}
	set.m = make(map[string]interface{})

	return set
}

// Add adds a new string to the set.
//
// Can be called multiple times on the same string.
func (set *StringSet) Add(str string) {
	set.m[str] = setMember
}

// IsMember tells you if the given string belongs to the set.
func (set *StringSet) IsMember(str string) bool {
	_, exists := set.m[str]

	return exists
}

// Remove deletes a string from the set.
//
// Can be called multiple times on the same string.
func (set *StringSet) Remove(str string) {
	delete(set.m, str)
}

// Members returns a slice with all the current members of the set.
func (set *StringSet) Members() []string {
	keys := make([]string, len(set.m))[:0]

	for k := range set.m {
		keys = append(keys, k)
	}

	return keys
}
