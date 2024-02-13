package registry

import (
	"encoding/json"
	"os"
)

const (
	// DefaultFileMode represents the default file permission mode
	defaultFileMode int = 0644
)

type StringRegistry struct {
	registryPath string
	registry     map[string]bool
}

// Creates and returns a string registry, using the local registry path as a
// persistent store
func NewStringRegistry(registryPath string) *StringRegistry {
	registry := &StringRegistry{
		registryPath: registryPath,
	}
	registry.loadRegistryState()

	return registry
}

// MarkProcessed marks the input item registered and saves the new registry state
func (sr *StringRegistry) Register(item string) {
	sr.registry[item] = true
	sr.saveRegistryState()
}

// IsRegistered returns whether or not the input item is registered
func (sr *StringRegistry) IsRegistered(item string) bool {
	return sr.registry[item]
}

// loadRegistryState load the StringRegistry's registry based on the
// registry's persistent store.
func (sr *StringRegistry) loadRegistryState() error {
	if _, err := os.Stat(sr.registryPath); os.IsNotExist(err) {
		sr.registry = make(map[string]bool)
		return nil
	}

	data, err := os.ReadFile(sr.registryPath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &sr.registry)
	if err != nil {
		return err
	}
	return nil
}

// saveRegistryState saves the registry's state to persistent storage
func (sr *StringRegistry) saveRegistryState() error {
	data, err := json.Marshal(sr.registry)
	if err != nil {
		return err
	}

	err = os.WriteFile(sr.registryPath, data, os.FileMode(defaultFileMode))
	if err != nil {
		return err
	}

	return nil
}
