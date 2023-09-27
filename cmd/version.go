package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/updater"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the current version & update information",
	Long:  `Display the current version & update information (if available).`,
	RunE: func(cmd *cobra.Command, args []string) error {

		updater.AllowPrereleases = true

		update, _ := cmd.Flags().GetBool("update")

		if update {
			return updateApp()
		}

		fmt.Printf("%s %s compiled with %s on %s/%s\n",
			os.Args[0], config.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

		latest, _, _, err := updater.GithubLatest(config.Repo, config.RepoBinaryName)
		if err == nil && updater.GreaterThan(latest, config.Version) {
			fmt.Printf(
				"\nUpdate available: %s\nRun `%s version -u` to update (requires read/write access to install directory).\n",
				latest,
				os.Args[0],
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().
		BoolP("update", "u", false, "update to latest version")
}

func updateApp() error {
	rel, err := updater.GithubUpdate(config.Repo, config.RepoBinaryName, config.Version)
	if err != nil {
		return err
	}

	fmt.Printf("Updated %s to version %s\n", os.Args[0], rel)
	return nil
}
