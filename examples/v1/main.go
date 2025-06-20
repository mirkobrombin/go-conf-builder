package main

import (
	"fmt"

	"github.com/mirkobrombin/go-conf-builder/v1/conf"
)

func main() {
	cfg := conf.New()
	cfg.SetConfigName("config.yaml")
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("./examples/v1")
	cfg.SetEnvPrefix("MYAPP")
	cfg.AutomaticEnv()
	cfg.SetDefault("debug", false)
	cfg.SetDefault("port", 8080)
	cfg.SetDefault("log_level", "info")
	cfg.SetDefault("timeout", 30)
	if err := cfg.ReadInConfig(); err != nil {
		fmt.Println("Config error:", err)
	}
	fmt.Println("Debug:", cfg.GetBool("debug"))
	fmt.Println("Port:", cfg.GetInt("port"))
	fmt.Println("Log Level:", cfg.GetString("log_level"))
	fmt.Println("Timeout:", cfg.GetInt("timeout"))
}
