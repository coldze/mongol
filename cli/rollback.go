package cli

import (
	"github.com/coldze/mongol/commands"
	"github.com/coldze/primitives/logs"
	"github.com/spf13/cobra"
)

func addRollbackCommand(rootCmd *cobra.Command, logger logs.Logger) {
	var path string
	var limit int64
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback migrations",
		Long:  "Rollback migrations",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logger.Infof("Migration path: '%v'", path)
			err := commands.Rollback(path, limit, logger)
			if err != nil {
				panic(err)
			}
		},
	}
	cmd.Flags().StringVarP(&path, "path", "t", "./changelog.json", "full path to migrations' map. Default: ./changelog.json")
	cmd.Flags().Int64VarP(&limit, "count", "c", -1, "limit amount of changes applied in a run. Values equal or below 0 are treated as 'apply everything'. Default: -1")
	rootCmd.AddCommand(cmd)
}
