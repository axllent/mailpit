package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/axllent/mailpit/updater"
	"github.com/spf13/cobra"
)

var (
	// Version is the default application version, updated on release
	Version = "dev"

	// Repo on Github for updater
	Repo = "axllent/mailpit"

	// RepoBinaryName on Github for updater
	RepoBinaryName = "mailpit"
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
			os.Args[0], Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

		latest, _, _, err := updater.GithubLatest(Repo, RepoBinaryName)
		if err == nil && updater.GreaterThan(latest, Version) {
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
	rel, err := updater.GithubUpdate(Repo, RepoBinaryName, Version)
	if err != nil {
		return err
	}

	fmt.Printf("Updated %s to version %s\n", os.Args[0], rel)
	return nil
}
