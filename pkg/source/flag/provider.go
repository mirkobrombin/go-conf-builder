package flag

import (
	"context"
	"flag"
	"strings"

	"github.com/mirkobrombin/go-conf-builder/v2/pkg/source"
)

// Provider implements source.Provider for standard flags.
type Provider struct{}

// New creates a new Flag provider.
func New() source.Provider {
	return &Provider{}
}

// Name returns "flag".
func (p *Provider) Name() string {
	return "flag"
}

// Load inspects all visited flags from the standard library `flag` package.
func (p *Provider) Load(ctx context.Context) (map[string]any, error) {
	data := make(map[string]any)

	flag.Visit(func(f *flag.Flag) {
		key := strings.ToLower(f.Name)
		key = strings.ReplaceAll(key, "-", "_")
		data[key] = f.Value.String()
	})

	return data, nil
}
