package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mirkobrombin/go-conf-builder/v2/pkg/loader"
	"github.com/mirkobrombin/go-conf-builder/v2/pkg/source/env"
	flagSource "github.com/mirkobrombin/go-conf-builder/v2/pkg/source/flag"
)

type Config struct {
	AppName string `conf:"env:APP_NAME,default:MyGoApp"`
	Port    int    `conf:"env:PORT,flag:port,default:8080"`
	Debug   bool   `conf:"env:DEBUG,flag:debug,default:false"`
}

func main() {
	flag.Int("port", 0, "Server Port")
	flag.Bool("debug", false, "Enable Debug Mode")
	flag.Parse()

	l := loader.New(
		env.New("APP"),
		flagSource.New(),
	)

	cfg := &Config{}

	if err := l.Load(context.Background(), cfg); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded Config: %+v\n", cfg)
}
