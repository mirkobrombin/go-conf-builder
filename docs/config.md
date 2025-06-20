# Configuration

Go Conf Builder provides a simple configuration loader inspired by Viper. It can read JSON, YAML, TOML, INI and XML files and merge them with environment variables.

## Basic Usage

```go
cfg := conf.New()
cfg.SetConfigName("config")
cfg.AddConfigPath(".")
cfg.SetEnvPrefix("MYAPP")
cfg.AutomaticEnv()
if err := cfg.ReadInConfig(); err != nil {
    fmt.Println("error loading config:", err)
}
```

Values are accessed through the various `Get` helpers:

```go
port := cfg.GetInt("port")
debug := cfg.GetBool("debug")
```

Environment variables override file values and defaults. Keys are automatically converted to upper case and prefixed.

## Watching for Changes

```go
cfg.OnConfigChange(func() {
    fmt.Println("config changed")
})
cfg.WatchConfig()
```

