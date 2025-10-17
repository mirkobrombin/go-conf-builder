package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
)

// Config provides configuration handling similar to Viper.
type Config struct {
	mu          sync.RWMutex
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
	watcherDone chan struct{}
	loaders     map[string]Loader
}

// New creates a new Config instance.
func New() *Config {
	c := &Config{
		defaults:    make(map[string]any),
		values:      make(map[string]any),
		envBindings: make(map[string]string),
		cfgPaths:    []string{"."},
	}
	c.loaders = defaultLoaders()
	return c
}

// SetEnvPrefix sets a prefix for environment variables.
func (c *Config) SetEnvPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.envPrefix = prefix
}

// AutomaticEnv enables automatic environment variable lookup.
func (c *Config) AutomaticEnv() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.automatic = true
}

// BindEnv binds a configuration key to a specific environment variable.
func (c *Config) BindEnv(key, env string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.envBindings[key] = env
}

// SetDefault sets a default value for a key.
func (c *Config) SetDefault(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaults[key] = value
}

// SetConfigName defines the base name of the config file.
func (c *Config) SetConfigName(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfgName = name
}

// SetConfigType sets the expected config file extension.
func (c *Config) SetConfigType(t string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfgType = strings.ToLower(t)
}

// AddConfigPath adds a path to search for the config file.
func (c *Config) AddConfigPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfgPaths = append(c.cfgPaths, path)
}

// SetConfigFile explicitly sets the config file path.
func (c *Config) SetConfigFile(file string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.file = file
}

// ReadInConfig reads the configuration file and merges values.
func (c *Config) ReadInConfig() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readInConfigLocked()
}

// ReadConfig reads configuration data from the provided reader and merges it.
func (c *Config) ReadConfig(r io.Reader) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfgType == "" {
		return errors.New("config type not set")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	parsed, err := c.decodeConfig(data, c.cfgType)
	if err != nil {
		return err
	}
	c.mergeConfigMapLocked(parsed)
	return nil
}

// MergeConfigMap merges the provided map into the current configuration.
func (c *Config) MergeConfigMap(data map[string]any) {
	if data == nil {
		return
	}
	normalized := normalizeLoadedMap(cloneMap(data))
	if normalized == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mergeConfigMapLocked(normalized)
}

func (c *Config) readInConfigLocked() error {
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
	parsed, err := c.decodeConfig(data, strings.TrimPrefix(strings.ToLower(filepath.Ext(c.file)), "."))
	if err != nil {
		return err
	}
	c.values = make(map[string]any)
	c.mergeConfigMapLocked(parsed)
	return nil
}

func (c *Config) mergeConfigMapLocked(data map[string]any) {
	if data == nil {
		return
	}
	if c.values == nil {
		c.values = data
		return
	}
	mergeMaps(c.values, data)
}

// RegisterLoader registers or replaces the loader responsible for the provided extension.
// The extension can optionally include a leading dot and is normalized to lower case.
func (c *Config) RegisterLoader(ext string, loader Loader) {
	normalized := strings.ToLower(strings.TrimPrefix(ext, "."))
	if normalized == "" || loader == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaders == nil {
		c.loaders = make(map[string]Loader)
	}
	c.loaders[normalized] = loader
}

func (c *Config) decodeConfig(data []byte, format string) (map[string]any, error) {
	format = strings.ToLower(strings.TrimPrefix(format, "."))
	if format == "" {
		return nil, errors.New("unsupported config file type")
	}
	loader, ok := c.loaders[format]
	if !ok || loader == nil {
		return nil, errors.New("unsupported config file type")
	}
	values, err := loader.Load(data)
	if err != nil {
		return nil, err
	}
	return normalizeLoadedMap(values), nil
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = cloneValue(v)
	}
	return dst
}

func cloneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return cloneMap(val)
	case map[any]any:
		copied := make(map[any]any, len(val))
		for k, item := range val {
			copied[k] = cloneValue(item)
		}
		return copied
	case []any:
		copied := make([]any, len(val))
		for i, item := range val {
			copied[i] = cloneValue(item)
		}
		return copied
	case []map[string]any:
		copied := make([]any, len(val))
		for i, item := range val {
			copied[i] = cloneValue(item)
		}
		return copied
	case []map[any]any:
		copied := make([]any, len(val))
		for i, item := range val {
			copied[i] = cloneValue(item)
		}
		return copied
	default:
		return v
	}
}

func mergeMaps(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}
	for k, v := range src {
		if existing, ok := dst[k]; ok {
			existingMap, existingIsMap := existing.(map[string]any)
			newMap, newIsMap := v.(map[string]any)
			if existingIsMap && newIsMap {
				dst[k] = mergeMaps(existingMap, newMap)
				continue
			}
		}
		dst[k] = v
	}
	return dst
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
	if v, ok := fetchValue(c.values, key); ok {
		return v, true
	}
	if v, ok := c.getEnv(key); ok {
		return v, true
	}
	return fetchValue(c.defaults, key)
}

func fetchValue(data map[string]any, key string) (any, bool) {
	if data == nil {
		return nil, false
	}
	if v, ok := data[key]; ok {
		return v, true
	}
	parts := strings.Split(key, ".")
	var current any = data
	for _, part := range parts {
		switch node := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = node[part]
			if !ok {
				return nil, false
			}
		case map[string]string:
			var ok bool
			current, ok = node[part]
			if !ok {
				return nil, false
			}
		case map[any]any:
			var ok bool
			current, ok = node[part]
			if !ok {
				return nil, false
			}
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			current = node[idx]
		case []map[string]any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			current = node[idx]
		case []map[interface{}]any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			current = node[idx]
		default:
			return nil, false
		}
	}
	return current, true
}

func normalizeLoadedMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	for k, v := range values {
		values[k] = normalizeValue(v)
	}
	return values
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			v[key] = normalizeValue(item)
		}
		return v
	case map[any]any:
		converted := make(map[string]any, len(v))
		for key, item := range v {
			converted[fmt.Sprint(key)] = normalizeValue(item)
		}
		return converted
	case []any:
		for i, item := range v {
			v[i] = normalizeValue(item)
		}
		return v
	case []map[string]any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeValue(item)
		}
		return result
	case []map[any]any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeValue(item)
		}
		return result
	default:
		return value
	}
}

// OnConfigChange sets a callback for configuration changes.
func (c *Config) OnConfigChange(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onChange = fn
}

// WatchConfig starts watching the config file for changes.
func (c *Config) WatchConfig() error {
	c.mu.Lock()
	if c.file == "" {
		c.mu.Unlock()
		return nil
	}
	if c.watcher != nil {
		c.mu.Unlock()
		return nil
	}
	file := c.file
	c.mu.Unlock()

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	done := make(chan struct{})

	c.mu.Lock()
	if c.file == "" {
		c.mu.Unlock()
		w.Close()
		return nil
	}
	if c.watcher != nil {
		c.mu.Unlock()
		w.Close()
		return nil
	}
	c.watcher = w
	c.watcherDone = done
	file = c.file
	c.mu.Unlock()

	go func(watcher *fsnotify.Watcher) {
		defer close(done)
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Op&fsnotify.Write == fsnotify.Write {
					if err := c.ReadInConfig(); err != nil {
						log.Printf("conf: failed to reload config: %v", err)
						continue
					}
					c.mu.RLock()
					callback := c.onChange
					c.mu.RUnlock()
					if callback != nil {
						callback()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if err != nil {
					log.Printf("conf: watcher error: %v", err)
				}
			}
		}
	}(w)

	return w.Add(file)
}

// Close releases resources associated with the watcher and resets its state.
func (c *Config) Close() error {
	c.mu.Lock()
	if c.watcher == nil {
		c.mu.Unlock()
		return nil
	}
	w := c.watcher
	done := c.watcherDone
	c.watcher = nil
	c.watcherDone = nil
	c.mu.Unlock()
	err := w.Close()
	if done != nil {
		<-done
	}
	return err
}

// GetString returns a string value for the key.
func (c *Config) GetString(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
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
	c.mu.RLock()
	defer c.mu.RUnlock()
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
	c.mu.RLock()
	defer c.mu.RUnlock()
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

// GetFloat64 returns a float64 value for the key. When the stored value is not
// compatible with a floating point representation, it falls back to 0.
func (c *Config) GetFloat64(key string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		case json.Number:
			f, _ := val.Float64()
			return f
		case string:
			f, _ := strconv.ParseFloat(val, 64)
			return f
		}
	}
	return 0
}

// GetDuration returns a time.Duration value for the key. Strings are parsed
// using time.ParseDuration, numeric values are treated as nanoseconds, and
// incompatible values yield 0.
func (c *Config) GetDuration(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case time.Duration:
			return val
		case int:
			return time.Duration(val)
		case int64:
			return time.Duration(val)
		case float64:
			return time.Duration(val)
		case string:
			d, err := time.ParseDuration(val)
			if err == nil {
				return d
			}
		}
	}
	return 0
}

// GetStringSlice returns a []string value for the key. Non compatible values
// result in an empty slice.
func (c *Config) GetStringSlice(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.get(key); ok {
		if res := toStringSlice(v); res != nil {
			return res
		}
	}
	return []string{}
}

// GetIntSlice returns a []int value for the key. Non convertible values result
// in an empty slice.
func (c *Config) GetIntSlice(key string) []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.get(key); ok {
		switch slice := v.(type) {
		case []int:
			return append([]int(nil), slice...)
		case []any:
			result := make([]int, 0, len(slice))
			for _, item := range slice {
				switch val := item.(type) {
				case int:
					result = append(result, val)
				case int64:
					result = append(result, int(val))
				case float64:
					result = append(result, int(val))
				case string:
					i, err := strconv.Atoi(val)
					if err != nil {
						return []int{}
					}
					result = append(result, i)
				default:
					return []int{}
				}
			}
			return result
		}
	}
	return []int{}
}

// GetStringMap returns a map[string]any value for the key. When the value is
// not a compatible map, it returns an empty map.
func (c *Config) GetStringMap(key string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case map[string]any:
			return cloneMap(val)
		case map[string]string:
			res := make(map[string]any, len(val))
			for k, item := range val {
				res[k] = item
			}
			return res
		case map[any]any:
			res := make(map[string]any, len(val))
			for k, item := range val {
				res[fmt.Sprint(k)] = item
			}
			return res
		}
	}
	return map[string]any{}
}

// GetStringMapString returns a map[string]string value for the key. On
// incompatible types, it returns an empty map.
func (c *Config) GetStringMapString(key string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case map[string]string:
			copy := make(map[string]string, len(val))
			for k, item := range val {
				copy[k] = item
			}
			return copy
		case map[string]any:
			res := make(map[string]string, len(val))
			for k, item := range val {
				res[k] = stringify(item)
			}
			return res
		case map[any]any:
			res := make(map[string]string, len(val))
			for k, item := range val {
				res[fmt.Sprint(k)] = stringify(item)
			}
			return res
		}
	}
	return map[string]string{}
}

// GetStringMapStringSlice returns a map[string][]string for the key. When the
// value cannot be converted, an empty map is returned.
func (c *Config) GetStringMapStringSlice(key string) map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.get(key); ok {
		switch val := v.(type) {
		case map[string][]string:
			res := make(map[string][]string, len(val))
			for k, item := range val {
				copy := append([]string(nil), item...)
				res[k] = copy
			}
			return res
		case map[string]any:
			res := make(map[string][]string, len(val))
			for k, item := range val {
				res[k] = toStringSlice(item)
			}
			return res
		case map[any]any:
			res := make(map[string][]string, len(val))
			for k, item := range val {
				res[fmt.Sprint(k)] = toStringSlice(item)
			}
			return res
		}
	}
	return map[string][]string{}
}

// Unmarshal decodes the configuration at the provided key into the given
// output struct. Nested maps are projected using mapstructure with weak typing.
func (c *Config) Unmarshal(key string, out any) error {
	if out == nil {
		return errors.New("conf: output cannot be nil")
	}
	var (
		data any
		ok   bool
	)
	c.mu.RLock()
	if key == "" {
		if c.values != nil {
			data = cloneMap(c.values)
			ok = true
		}
	} else {
		if v, exists := c.get(key); exists {
			data = cloneValue(v)
			ok = true
		}
	}
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("conf: key %q not found", key)
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          "mapstructure",
		Result:           out,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	})
	if err != nil {
		return err
	}
	return decoder.Decode(data)
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

func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return append([]string(nil), val...)
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			result = append(result, stringify(item))
		}
		return result
	case string:
		if val == "" {
			return []string{}
		}
		parts := strings.Split(val, ",")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		return parts
	default:
		return nil
	}
}
