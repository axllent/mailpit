package cmd

import (
	"os"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/spf13/cobra"
)

// reindexCmd represents the reindex command
var reindexCmd = &cobra.Command{
	Use:   "reindex <database>",
	Short: "Reindex the database",
	Long: `This will reindex all messages in the entire database.

If you have several thousand messages in your mailbox, then it is advised to shut down
Mailpit while you reindex as this process will likely result in database locking issues.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config.Database = args[0]
		config.MaxMessages = 0

		if err := storage.InitDB(); err != nil {
			logger.Log().Error(err)
			os.Exit(1)
		}

		storage.ReindexAll()
	},
}

func init() {
	rootCmd.AddCommand(reindexCmd)
}
