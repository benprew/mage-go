package catalog

import "sync"

var (
	mu            sync.Mutex
	globalCatalog = newCatalog()
)

// RegisterSet registers a set's card data into the global catalog.
// code is the set code (e.g. "ATQ"), name is the display name (e.g. "Antiquities"),
// and data is the raw JSON card data.
func RegisterSet(code, name string, data []byte) {
	mu.Lock()
	defer mu.Unlock()
	if err := globalCatalog.addSetFromBytes(code, name, data); err != nil {
		panic("catalog: failed to load set " + code + ": " + err.Error())
	}
}

// Global returns the shared catalog containing all registered sets.
func Global() *Catalog {
	return globalCatalog
}
