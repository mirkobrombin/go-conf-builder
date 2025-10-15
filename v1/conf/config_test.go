package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultsAndEnv(t *testing.T) {
	c := New()
	c.SetEnvPrefix("APP")
	c.SetDefault("port", 8080)
	if c.GetInt("port") != 8080 {
		t.Fatalf("expected default 8080")
	}

	os.Setenv("APP_PORT", "9001")
	defer os.Unsetenv("APP_PORT")
	if c.GetInt("port") != 9001 {
		t.Fatalf("env variable should override default")
	}
}

func TestReadConfigFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "cfg*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("debug: true\nvalue: 42\n")
	tmp.Close()

	c := New()
	c.SetConfigFile(tmp.Name())
	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if !c.GetBool("debug") {
		t.Fatalf("expected debug true")
	}
	if c.GetInt("value") != 42 {
		t.Fatalf("expected value 42")
	}
}

func TestConfigNameAndPath(t *testing.T) {
	tmp, err := os.CreateTemp("", "conf*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("port=9000\n")
	tmp.Close()

	dir := filepath.Dir(tmp.Name())
	base := filepath.Base(tmp.Name())
	name := strings.TrimSuffix(base, filepath.Ext(base))

	c := New()
	c.SetConfigName(name)
	c.SetConfigType("toml")
	c.AddConfigPath(dir)
	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if c.GetInt("port") != 9000 {
		t.Fatalf("expected port 9000")
	}
}

func TestNestedConfigAccess(t *testing.T) {
	tests := []struct {
		name    string
		ext     string
		content string
	}{
		{
			name:    "json",
			ext:     ".json",
			content: `{"database":{"hosts":["db1","db2"],"port":5432}}`,
		},
		{
			name:    "yaml",
			ext:     ".yaml",
			content: "database:\n  hosts:\n    - db1\n    - db2\n  port: 5432\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp, err := os.CreateTemp("", "cfg*"+tt.ext)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmp.Name())
			if _, err := tmp.WriteString(tt.content); err != nil {
				t.Fatal(err)
			}
			tmp.Close()

			c := New()
			c.SetConfigFile(tmp.Name())
			if err := c.ReadInConfig(); err != nil {
				t.Fatal(err)
			}

			if got := c.GetString("database.hosts.0"); got != "db1" {
				t.Fatalf("expected first host db1, got %q", got)
			}
			if got := c.GetInt("database.port"); got != 5432 {
				t.Fatalf("expected port 5432, got %d", got)
			}
		})
	}
}

func TestNestedEnvOverride(t *testing.T) {
	tmp, err := os.CreateTemp("", "cfg*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString("database:\n  hosts:\n    - db1\n    - db2\n"); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	c := New()
	c.SetEnvPrefix("APP")
	c.AutomaticEnv()
	c.SetConfigFile(tmp.Name())
	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	os.Setenv("APP_DATABASE_HOSTS_1", "db-override")
	defer os.Unsetenv("APP_DATABASE_HOSTS_1")

	if got := c.GetString("database.hosts.1"); got != "db-override" {
		t.Fatalf("expected env override db-override, got %q", got)
	}
}

func TestReadInConfigRemovesMissingKeys(t *testing.T) {
	tmp, err := os.CreateTemp("", "cfg*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	initial := "keep: 1\nremove: 2\n"
	if err := os.WriteFile(tmp.Name(), []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	c := New()
	c.SetConfigFile(tmp.Name())
	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	if got := c.GetInt("keep"); got != 1 {
		t.Fatalf("expected keep=1, got %d", got)
	}
	if got := c.GetInt("remove"); got != 2 {
		t.Fatalf("expected remove=2, got %d", got)
	}

	updated := "keep: 3\n"
	if err := os.WriteFile(tmp.Name(), []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	if got := c.GetInt("keep"); got != 3 {
		t.Fatalf("expected keep=3, got %d", got)
	}
	if _, ok := c.values["remove"]; ok {
		t.Fatalf("expected remove key to be cleared")
	}
	if got := c.GetInt("remove"); got != 0 {
		t.Fatalf("expected remove=0 after deletion, got %d", got)
	}
}

func TestMergeSequenceWithMapsAndReaders(t *testing.T) {
	c := New()
	c.MergeConfigMap(map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	})

	c.SetConfigType("yaml")
	reader := strings.NewReader("server:\n  port: 9000\n  ssl: true\nfeature:\n  enabled: true\n")
	if err := c.ReadConfig(reader); err != nil {
		t.Fatal(err)
	}

	c.MergeConfigMap(map[string]any{
		"feature": map[string]any{
			"name":    "beta",
			"enabled": false,
		},
		"extra": "value",
	})

	if got := c.GetString("server.host"); got != "localhost" {
		t.Fatalf("expected server.host to remain localhost, got %q", got)
	}
	if got := c.GetInt("server.port"); got != 9000 {
		t.Fatalf("expected server.port=9000, got %d", got)
	}
	if !c.GetBool("server.ssl") {
		t.Fatalf("expected server.ssl true")
	}
	if c.GetBool("feature.enabled") {
		t.Fatalf("expected feature.enabled false after override")
	}
	if got := c.GetString("feature.name"); got != "beta" {
		t.Fatalf("expected feature.name=beta, got %q", got)
	}
	if got := c.GetString("extra"); got != "value" {
		t.Fatalf("expected extra=value, got %q", got)
	}
}

func TestGetters(t *testing.T) {
	c := New()
	c.MergeConfigMap(map[string]any{
		"number":   "3.14",
		"duration": "5s",
		"strings":  []any{"a", 2, true},
		"ints":     []any{"1", 2, 3.0},
		"mapping": map[string]any{
			"key":  12,
			"list": []any{"x", "y"},
		},
	})

	if got := c.GetFloat64("number"); got != 3.14 {
		t.Fatalf("expected float64 3.14, got %f", got)
	}
	if got := c.GetDuration("duration"); got != 5*time.Second {
		t.Fatalf("expected duration 5s, got %s", got)
	}
	if got := c.GetStringSlice("strings"); len(got) != 3 || got[0] != "a" || got[1] != "2" || got[2] != "true" {
		t.Fatalf("unexpected string slice %#v", got)
	}
	if got := c.GetIntSlice("ints"); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("unexpected int slice %#v", got)
	}

	strMap := c.GetStringMap("mapping")
	if stringify(strMap["key"]) != "12" {
		t.Fatalf("expected mapping.key to stringify to 12, got %v", strMap["key"])
	}
	mapString := c.GetStringMapString("mapping")
	if mapString["key"] != "12" {
		t.Fatalf("expected string map key to be 12, got %q", mapString["key"])
	}
	mapSlice := c.GetStringMapStringSlice("mapping")
	if len(mapSlice["list"]) != 2 || mapSlice["list"][0] != "x" || mapSlice["list"][1] != "y" {
		t.Fatalf("expected list to be [x y], got %#v", mapSlice["list"])
	}
}

type appConfig struct {
	Server struct {
		Host    string        `mapstructure:"host"`
		Port    int           `mapstructure:"port"`
		Timeout time.Duration `mapstructure:"timeout"`
	} `mapstructure:"server"`
	Feature struct {
		Enabled bool `mapstructure:"enabled"`
	}
}

func TestUnmarshal(t *testing.T) {
	c := New()
	c.MergeConfigMap(map[string]any{
		"server": map[string]any{
			"host":    "localhost",
			"port":    8080,
			"timeout": "10s",
		},
		"feature": map[string]any{
			"enabled": true,
		},
	})

	var cfg appConfig
	if err := c.Unmarshal("", &cfg); err != nil {
		t.Fatalf("unexpected error unmarshalling root: %v", err)
	}
	if cfg.Server.Host != "localhost" || cfg.Server.Port != 8080 || cfg.Server.Timeout != 10*time.Second {
		t.Fatalf("unexpected server config %+v", cfg.Server)
	}
	if !cfg.Feature.Enabled {
		t.Fatalf("expected feature.enabled true")
	}

	var server struct {
		Host string `mapstructure:"host"`
	}
	if err := c.Unmarshal("server", &server); err != nil {
		t.Fatalf("unexpected error unmarshalling server: %v", err)
	}
	if server.Host != "localhost" {
		t.Fatalf("expected server host localhost, got %q", server.Host)
	}

	if err := c.Unmarshal("missing", &server); err == nil {
		t.Fatalf("expected error for missing key")
	}
}
