# Go Conf Builder

> [!CAUTION]
> go-conf-builder is now part of the [go-foundation](https://github.com/mirkobrombin/go-foundation) framework. The v1.0.0 release mirrors go-conf-builder v2.0.1, but future versions may introduce breaking changes. Please migrate your project.

A **declarative**, struct-tag based configuration loader for Go.

## Features

- **Declarative Configuration:** Define your config structure using struct tags (`conf`).
- **Multiple Sources:** Load from Environment Variables, Command-line Flags, and more.
- **Type-Safe:** Automatically binds values to basic types (`int`, `bool`, `string`, `float64`, etc.).
- **Pluggable Architecture:** Easily add new configuration sources by implementing the `Provider` interface.

## Installation

```bash
go get github.com/mirkobrombin/go-conf-builder/v2
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/mirkobrombin/go-conf-builder/v2/pkg/loader"
    "github.com/mirkobrombin/go-conf-builder/v2/pkg/source/env"
    "github.com/mirkobrombin/go-conf-builder/v2/pkg/source/flag"
)

type Config struct {
    AppName string `conf:"env:APP_NAME,default:MyGoApp"`
    Port    int    `conf:"env:PORT,flag:port,default:8080"`
    Debug   bool   `conf:"env:DEBUG,flag:debug,default:false"`
}

func main() {
    // Define sources
    l := loader.New(
        env.New("APP"),
        flag.New(),
    )

    // Load into struct
    cfg := &Config{}
    if err := l.Load(context.Background(), cfg); err != nil {
        panic(err)
    }

    fmt.Printf("Loaded: %+v\n", cfg)
}
```

## Documentation

- [Configuration Guide](docs/config.md)

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
