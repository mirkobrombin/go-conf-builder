# Configuration

Go Conf Builder provides a simple, extensible configuration loader inspired by Viper.
It can read **JSON, YAML, TOML, INI, and XML** files, merge them with **environment variables**, and even support **custom configuration formats** through pluggable loaders.

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

Values can be accessed via typed getters:

```go
port := cfg.GetInt("port")
debug := cfg.GetBool("debug")
```

Environment variables override both file values and defaults.
Keys are automatically converted to uppercase and prefixed (e.g. `MYAPP_PORT`).

## Watching for Changes

```go
cfg.OnConfigChange(func() {
    fmt.Println("config changed")
})
cfg.WatchConfig()
```

`WatchConfig` uses **fsnotify** to monitor changes in the loaded configuration file and automatically trigger the registered callback.

## Supported Formats

By default, the following formats are supported:

| Extension        | Loader       | Package Used                 |
| ---------------- | ------------ | ---------------------------- |
| `.json`          | `JSONLoader` | `encoding/json`              |
| `.yaml` / `.yml` | `YAMLLoader` | `gopkg.in/yaml.v3`           |
| `.toml`          | `TOMLLoader` | `github.com/BurntSushi/toml` |
| `.ini`           | `INILoader`  | `gopkg.in/ini.v1`            |
| `.xml`           | `XMLLoader`  | `encoding/xml`               |

Each loader decodes data into a `map[string]any`, allowing recursive merging and normalization.

## Custom Loaders

You can register your own loader for any file extension:

```go
type FakeLoader struct{}

func (FakeLoader) Load(data []byte) (map[string]any, error) {
    return map[string]any{"raw": strings.TrimSpace(string(data))}, nil
}

cfg := conf.New()
cfg.RegisterLoader("fake", FakeLoader{})
cfg.SetConfigFile("config.fake")
if err := cfg.ReadInConfig(); err != nil {
    panic(err)
}
fmt.Println(cfg.GetString("raw")) // prints decoded value
```

Extensions are normalized (case-insensitive, no leading dot).
If a custom loader is registered, it **replaces** the default one for that extension.

## Programmatic Reads

Besides reading from disk, configuration can be read directly from an `io.Reader`:

```go
cfg.SetConfigType("yaml")
data := strings.NewReader("key: value")
cfg.ReadConfig(data)
fmt.Println(cfg.GetString("key")) // → "value"
```

This is useful for loading from memory, embedded assets, or network responses.

## Thread Safety

All configuration access is **thread-safe**.
Internal state (defaults, values, watchers) is guarded by mutexes.

## Summary

* Supports JSON, YAML, TOML, INI, XML out of the box
* Allows **custom loaders** via `RegisterLoader`
* Supports **environment variable overrides**
* Detects **file changes** and triggers callbacks
* Provides **safe concurrent access**
* Works with **disk files** or **in-memory readers**

This makes `conf` a minimal yet flexible configuration system suited for both CLI tools and long-running services.
