package main

import (
	"os"

	"github.com/go-kratos/kratos/v2/log"

	_ "github.com/go-kratos/kratos/v2/config/file"
)

func main() {
	confPath := os.Getenv("CONF_PATH")
	if confPath == "" {
		confPath = "configs/billing-service.yaml"
	}

	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)

	app, cleanup, err := InitApp(confPath)
	if err != nil {
		log.Errorf("failed to create app: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		log.Errorf("failed to run app: %v", err)
		os.Exit(1)
	}
}
