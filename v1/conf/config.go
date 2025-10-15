package conf

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
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
	c.values = make(map[string]any)
	parsed, err := c.decodeConfig(data, strings.TrimPrefix(strings.ToLower(filepath.Ext(c.file)), "."))
	if err != nil {
		return err
	}
	c.MergeConfigMap(parsed)
	return nil
}

// ReadConfig reads configuration data from the provided reader and merges it.
func (c *Config) ReadConfig(r io.Reader) error {
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
	c.MergeConfigMap(parsed)
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
	if c.values == nil {
		c.values = normalized
		return
	}
	mergeMaps(c.values, normalized)
}

func (c *Config) decodeConfig(data []byte, format string) (map[string]any, error) {
	format = strings.ToLower(strings.TrimPrefix(format, "."))
	var err error
	values := make(map[string]any)
	switch format {
	case "json":
		err = json.Unmarshal(data, &values)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &values)
	case "toml":
		_, err = toml.Decode(string(data), &values)
	case "ini":
		var cfg *ini.File
		cfg, err = ini.Load(data)
		if err == nil {
			for k, v := range cfg.Section("").KeysHash() {
				values[k] = v
			}
		}
	case "xml":
		err = xml.Unmarshal(data, (*map[string]any)(&values))
	default:
		err = errors.New("unsupported config file type")
	}
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

// GetFloat64 returns a float64 value for the key. When the stored value is not
// compatible with a floating point representation, it falls back to 0.
func (c *Config) GetFloat64(key string) float64 {
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
	if key == "" {
		data = c.values
		ok = data != nil
	} else {
		data, ok = c.get(key)
	}
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
