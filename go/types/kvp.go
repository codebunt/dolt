package types

// KVP is a simple key value pair
type KVP struct {
	// Key is the key
	Key LesserValuable

	// Val is the value
	Val Valuable
}

// KVPSlice is a slice of KVPs that implements sort.Interface
type KVPSlice []KVP

// Len returns the size of the slice
func (kvps KVPSlice) Len() int {
	return len(kvps)
}

// Less returns a bool representing whether the key at index i is less than the key at index j
func (kvps KVPSlice) Less(i, j int) bool {
	return kvps[i].Key.Less(kvps[j].Key)
}

// Swap swaps the KVP at index i with the KVP at index j
func (kvps KVPSlice) Swap(i, j int) {
	kvps[i], kvps[j] = kvps[j], kvps[i]
}

// KVPIterator is an interface for iterating over KVPs.  There are implementations for KVPSlice, KVPCollection, and
// for two KVPCollection instances which merges as it iterates
type KVPIterator interface {
	Next() *KVP
	Peek() *KVP
}

// IsInOrder iterates over every value and validates that they are returned in key order.  This is intended for testing.
func IsInOrder(itr KVPIterator) (bool, int) {
	prev := itr.Next()

	if prev == nil {
		return true, 0
	}

	count := 1

	for {
		var curr *KVP
		curr = itr.Next()

		if curr == nil {
			break
		} else if curr.Key.Less(prev.Key) {
			return false, count
		}

		count++
		prev = curr
	}

	return true, count
}
