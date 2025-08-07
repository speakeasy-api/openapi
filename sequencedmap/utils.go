package sequencedmap

import "iter"

// Len returns the number of elements in the map. nil safe.
func Len[K comparable, V any](m *Map[K, V]) int {
	if m == nil {
		return 0
	}
	return len(m.l)
}

// From creates a new map from the given sequence.
func From[K comparable, V any](seq iter.Seq2[K, V]) *Map[K, V] {
	newMap := New[K, V]()

	for k, v := range seq {
		newMap.Set(k, v)
	}

	return newMap
}
