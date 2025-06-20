package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
