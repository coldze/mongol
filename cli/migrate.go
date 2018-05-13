package cli

import (
	"github.com/coldze/mongol/commands"
	"github.com/coldze/mongol/common/logs"
	"github.com/spf13/cobra"
)

func addMigrateCommand(rootCmd *cobra.Command, logger logs.Logger) {
	var path string
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run migrations",
		Long:  "Run migrations",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logger.Infof("Migration path: '%v'", path)
			err := commands.Migrate(path, logger)
			if err != nil {
				panic(err)
			}
		},
	}
	cmd.Flags().StringVarP(&path, "path", "t", "./changelog.json", "full path to migrations' map. Default: ./changelog.json")
	rootCmd.AddCommand(cmd)
}
