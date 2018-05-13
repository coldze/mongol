package main

import (
	"github.com/coldze/mongol/cli"
	"github.com/coldze/mongol/common/logs"
	"os"
)

func main() {
	logger := logs.NewStdLogger()
	app := cli.NewCliApp(logger)
	err := app.Run()
	if err == nil {
		return
	}
	logger.Fatalf("App run failed with error: %v", err)
	os.Exit(1)
}
