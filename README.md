# Go Conf Builder

Go Conf Builder is a lightweight configuration library inspired by Viper. It allows loading configuration values from various file formats and environment variables.

## Features

- Read JSON, YAML, TOML, INI and XML files
- Set default values for keys
- Bind environment variables with optional prefixes
- Automatic environment variable loading
- Watch configuration files for changes and react via callbacks

## Installation

```bash
go get github.com/mirkobrombin/go-conf-builder
```

## Basic Usage

```go
cfg := conf.New()
cfg.SetConfigName("config")
cfg.AddConfigPath(".")
cfg.SetEnvPrefix("MYAPP")
cfg.AutomaticEnv()
cfg.SetDefault("port", 8080)

if err := cfg.ReadInConfig(); err != nil {
    fmt.Println("error loading config:", err)
}

fmt.Println("port:", cfg.GetInt("port"))
```

See the [docs](docs/config.md) for more details.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
