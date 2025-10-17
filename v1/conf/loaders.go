package conf

import (
	"encoding/json"
	"encoding/xml"

	"github.com/BurntSushi/toml"
	ini "gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

// Loader defines the behavior for parsing configuration data into a map.
type Loader interface {
	Load(data []byte) (map[string]any, error)
}

// JSONLoader implements Loader for JSON documents.
type JSONLoader struct{}

// Load decodes JSON data into a map representation.
func (JSONLoader) Load(data []byte) (map[string]any, error) {
	values := make(map[string]any)
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

// YAMLLoader implements Loader for YAML documents.
type YAMLLoader struct{}

// Load decodes YAML data into a map representation.
func (YAMLLoader) Load(data []byte) (map[string]any, error) {
	values := make(map[string]any)
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

// TOMLLoader implements Loader for TOML documents.
type TOMLLoader struct{}

// Load decodes TOML data into a map representation.
func (TOMLLoader) Load(data []byte) (map[string]any, error) {
	values := make(map[string]any)
	if _, err := toml.Decode(string(data), &values); err != nil {
		return nil, err
	}
	return values, nil
}

// INILoader implements Loader for INI documents.
type INILoader struct{}

// Load decodes INI data into a map representation.
func (INILoader) Load(data []byte) (map[string]any, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(cfg.Section("").Keys()))
	for k, v := range cfg.Section("").KeysHash() {
		values[k] = v
	}
	return values, nil
}

// XMLLoader implements Loader for XML documents.
type XMLLoader struct{}

// Load decodes XML data into a map representation.
func (XMLLoader) Load(data []byte) (map[string]any, error) {
	values := make(map[string]any)
	if err := xml.Unmarshal(data, (*map[string]any)(&values)); err != nil {
		return nil, err
	}
	return values, nil
}

func defaultLoaders() map[string]Loader {
	return map[string]Loader{
		"json": JSONLoader{},
		"yaml": YAMLLoader{},
		"yml":  YAMLLoader{},
		"toml": TOMLLoader{},
		"ini":  INILoader{},
		"xml":  XMLLoader{},
	}
}
