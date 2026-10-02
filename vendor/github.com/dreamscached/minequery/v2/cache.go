package minequery

// Cache is a common interface for various cache implementations
// to provide abstract layer for other caching libraries.
type Cache interface {
	// Get retrieves value by cache key, either returning the value
	// itself and true as second return value or nil and false correspondingly.
	Get(string) (interface{}, bool)

	// SetDefault sets value by cache key with default TTL and expiration
	// configuration parameters.
	SetDefault(string, interface{})
}
