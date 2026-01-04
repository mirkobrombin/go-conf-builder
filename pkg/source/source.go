package source

import "context"

// Provider defines the interface for configuration sources.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// Load retrieves configuration key-value pairs.
	Load(ctx context.Context) (map[string]any, error)
}
