package main

import (
	"github.com/coldze/mongol/cli"
	"github.com/coldze/mongol/engine/decoding"
	"github.com/coldze/primitives/logs"
	"io/ioutil"
	"log"
	"os"
)

func Test() {
	data, err := ioutil.ReadFile("./src/0004_create_index_dashboard_assets_data.json")
	if err != nil {
		log.Fatalf("Failed to load file. Error: %v", err)
	}
	decoding.DecodeExt(data)
}

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
