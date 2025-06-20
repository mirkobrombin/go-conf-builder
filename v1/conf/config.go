package conf

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	ini "gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

// Config provides configuration handling similar to Viper.
type Config struct {
	defaults    map[string]any
	values      map[string]any
	envPrefix   string
	envBindings map[string]string
	cfgName     string
	cfgType     string
	cfgPaths    []string
	file        string
	automatic   bool
	watcher     *fsnotify.Watcher
	onChange    func()
}

// New creates a new Config instance.
func New() *Config {
	return &Config{
		defaults:    make(map[string]any),
		values:      make(map[string]any),
		envBindings: make(map[string]string),
		cfgPaths:    []string{"."},
	}
}

// SetEnvPrefix sets a prefix for environment variables.
func (c *Config) SetEnvPrefix(prefix string) { c.envPrefix = prefix }

// AutomaticEnv enables automatic environment variable lookup.
func (c *Config) AutomaticEnv() { c.automatic = true }

// BindEnv binds a configuration key to a specific environment variable.
func (c *Config) BindEnv(key, env string) { c.envBindings[key] = env }

// SetDefault sets a default value for a key.
func (c *Config) SetDefault(key string, value any) { c.defaults[key] = value }

// SetConfigName defines the base name of the config file.
func (c *Config) SetConfigName(name string) { c.cfgName = name }

// SetConfigType sets the expected config file extension.
func (c *Config) SetConfigType(t string) { c.cfgType = strings.ToLower(t) }

// AddConfigPath adds a path to search for the config file.
func (c *Config) AddConfigPath(path string) { c.cfgPaths = append(c.cfgPaths, path) }

// SetConfigFile explicitly sets the config file path.
func (c *Config) SetConfigFile(file string) { c.file = file }

// ReadInConfig reads the configuration file and merges values.
func (c *Config) ReadInConfig() error {
	if c.file == "" {
		if c.cfgName == "" {
			return nil
		}
		for _, p := range c.cfgPaths {
			name := filepath.Join(p, c.cfgName)
			if c.cfgType != "" {
				name += "." + c.cfgType
			}
			if _, err := os.Stat(name); err == nil {
				c.file = name
				break
			}
		}
		if c.file == "" {
			return os.ErrNotExist
		}
	}

	data, err := os.ReadFile(c.file)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(c.file))
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &c.values)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &c.values)
	case ".toml":
		_, err = toml.Decode(string(data), &c.values)
	case ".ini":
		cfg, e := ini.Load(data)
		if e == nil {
			m := cfg.Section("").KeysHash()
			for k, v := range m {
				c.values[k] = v
			}
		}
		err = e
	case ".xml":
		err = xml.Unmarshal(data, (*map[string]any)(&c.values))
	default:
		err = errors.New("unsupported config file type")
	}
	return err
}

func (c *Config) getEnv(key string) (string, bool) {
	env, ok := c.envBindings[key]
	if !ok {
		env = strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		if c.envPrefix != "" {
			env = c.envPrefix + "_" + env
		}
	}
	val, exists := os.LookupEnv(env)
	return val, exists
}

func (c *Config) get(key string) (any, bool) {
	if c.automatic {
		if v, ok := c.getEnv(key); ok {
			return v, true
		}
	}
	if v, ok := c.values[key]; ok {
		return v, true
	}
	if v, ok := c.getEnv(key); ok {
		return v, true
	}
	v, ok := c.defaults[key]
	return v, ok
}

// OnConfigChange sets a callback for configuration changes.
func (c *Config) OnConfigChange(fn func()) { c.onChange = fn }

// WatchConfig starts watching the config file for changes.
func (c *Config) WatchConfig() error {
	if c.file == "" {
		return nil
	}
	if c.watcher != nil {
		return nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	c.watcher = w
	go func() {
		for ev := range w.Events {
			if ev.Op&fsnotify.Write == fsnotify.Write {
				c.ReadInConfig()
				if c.onChange != nil {
					c.onChange()
				}
			}
		}
	}()
	return w.Add(c.file)
}

// GetString returns a string value for the key.
func (c *Config) GetString(key string) string {
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case string:
			return val
		default:
			return stringify(val)
		}
	}
	return ""
}

// GetInt returns an int value for the key.
func (c *Config) GetInt(key string) int {
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			i, _ := strconv.Atoi(val)
			return i
		}
	}
	return 0
}

// GetBool returns a boolean value for the key.
func (c *Config) GetBool(key string) bool {
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			b, _ := strconv.ParseBool(val)
			return b
		case int:
			return val != 0
		case float64:
			return val != 0
		}
	}
	return false
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
