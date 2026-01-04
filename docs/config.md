# Configuration Guide

Go Conf Builder (V2) uses a declarative approach based on struct tags and pluggable sources.

## Struct Tags

The `conf` tag is used to define how a field should be populated.

| Option | Description | Example |
|--------|-------------|---------|
| `env`  | Environment variable key | `env:PORT` |
| `flag` | Command-line flag name | `flag:port` |
| `default` | Default value if not found in any source | `default:8080` |

Multiple options are separated by commas:
```go
Port int `conf:"env:PORT,flag:port,default:8080"`
```

## Loader

The `Loader` is responsible for orchestrating the loading process from multiple sources.

```go
l := loader.New(sources...)
err := l.Load(ctx, &myConfig)
```

## Sources (Providers)

Sources implement the `Provider` interface and return a `map[string]any`.

### Environment Variables

```go
import "github.com/mirkobrombin/go-conf-builder/v2/pkg/source/env"

// Create with optional prefix
// If prefix is "APP", it will look for APP_PORT when tag is env:PORT
s := env.New("APP")
```

### Command-line Flags

```go
import "github.com/mirkobrombin/go-conf-builder/v2/pkg/source/flag"

// Reads from the standard library flag package
s := flag.New()
```

## Custom Sources

You can implement your own source by fulfilling the `Provider` interface:

```go
type Provider interface {
    Name() string
    Load(ctx context.Context) (map[string]any, error)
}
```

This allows loading configuration from files (YAML, JSON), remote stores (Etcd, Consul), or any other custom logic.

## Type Support

The loader automatically converts source strings to the following field types:
- `string`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `bool`
- `float32`, `float64`
