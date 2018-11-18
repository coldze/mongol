package main

import (
	"github.com/coldze/mongol/cli"
	"github.com/coldze/primitives/logs"
	"os"
)

func main() {
	logger := logs.NewStdLogger()
	app := cli.NewCliApp(logger)
	errC := app.Run()
	if errC == nil {
		return
	}
	logger.Fatalf("App run failed with error: %v", errC)
	os.Exit(1)
}
