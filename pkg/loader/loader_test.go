package loader_test

import (
	"context"
	"testing"

	"github.com/mirkobrombin/go-conf-builder/v2/pkg/loader"
	"github.com/mirkobrombin/go-conf-builder/v2/pkg/source/env"
)

func TestLoadIgnoresUnexportedFields(t *testing.T) {
	type Config struct {
		Host string `conf:"env:HOST"`
		port int    // unexported — must not panic
	}
	t.Setenv("HOST", "localhost")
	l := loader.New(env.New(""))
	var cfg Config
	if err := l.Load(context.Background(), &cfg); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Host != "localhost" {
		t.Fatalf("Host = %q, want %q", cfg.Host, "localhost")
	}
}
