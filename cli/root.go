package cli

import (
	"fmt"

	"github.com/coldze/primitives/custom_error"
	"github.com/coldze/primitives/logs"
	"github.com/spf13/cobra"
)

type App interface {
	Run() error
}

type Cli struct {
	rootCommand *cobra.Command
}

func (c *Cli) Run() (result error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		errValue, ok := r.(custom_error.CustomError)
		if ok {
			result = fmt.Errorf("Run failed. Error: %v", errValue)
			return
		}
		err, ok := r.(error)
		if ok {
			result = fmt.Errorf("Run failed. Error: %v", err)
			return
		}
		result = fmt.Errorf("Run failed. Unknown error: %+v", r)
	}()
	return c.rootCommand.Execute()
}

func NewCliApp(logger logs.Logger) App {

	rootCmd := &cobra.Command{
		Use:   "mongol",
		Short: "Migration tool for MongoDB.",
		Long:  "Migration tool for MongoDB.",
	}

	addMigrateCommand(rootCmd, logger)

	return &Cli{
		rootCommand: rootCmd,
	}
}
