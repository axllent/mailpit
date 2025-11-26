package cmd

import (
	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/dump"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/spf13/cobra"
)

// dumpCmd represents the dump command
var dumpCmd = &cobra.Command{
	Use:   "dump <database> <output-dir>",
	Short: "Dump all messages from a database to a directory",
	Long: `Dump all messages stored in Mailpit into a local directory as individual files.

The database can either be the database file (eg: --database /var/lib/mailpit/mailpit.db) or a
URL of a running Mailpit instance (eg: --http http://127.0.0.1/). If dumping over HTTP, the URL
should be the base URL of your running Mailpit instance, not the link to the API itself.`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		if err := dump.Sync(args[0]); err != nil {
			logger.Log().Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(dumpCmd)

	dumpCmd.Flags().SortFlags = false

	dumpCmd.Flags().StringVar(&config.Database, "database", config.Database, "Dump messages directly from a database file")
	dumpCmd.Flags().StringVar(&config.TenantID, "tenant-id", config.TenantID, "Database tenant ID to isolate data (optional)")
	dumpCmd.Flags().StringVar(&dump.URL, "http", dump.URL, "Dump messages via HTTP API (base URL of running Mailpit instance)")
	dumpCmd.Flags().BoolVarP(&logger.VerboseLogging, "verbose", "v", logger.VerboseLogging, "Verbose logging")
}
