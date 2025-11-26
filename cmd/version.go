package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/axllent/mailpit/config"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the current version & update information",
	Long:  `Display the current version & update information (if available).`,
	Run: func(cmd *cobra.Command, _ []string) {
		update, _ := cmd.Flags().GetBool("update")
		noReleaseCheck, _ := cmd.Flags().GetBool("no-release-check")

		if update {
			// Update the application
			rel, err := config.GHRUConfig.SelfUpdate()
			if err != nil {
				fmt.Printf("Error updating: %s\n", err)
				os.Exit(1)
			}

			fmt.Printf("Updated %s to version %s\n", os.Args[0], rel.Tag)
			os.Exit(0)
		}

		fmt.Printf("%s %s compiled with %s on %s/%s\n",
			os.Args[0], config.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

		if !noReleaseCheck {
			release, err := config.GHRUConfig.Latest()
			if err != nil {
				fmt.Printf("Error checking for latest release: %s\n", err)
				os.Exit(1)
			}

			// The latest version is the same version
			if release.Tag == config.Version {
				os.Exit(0)
			}

			// A newer release is available
			fmt.Printf(
				"\nUpdate available: %s\nRun `%s version -u` to update (requires read/write access to install directory).\n",
				release.Tag,
				os.Args[0],
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().
		BoolP("update", "u", false, "update to latest version")
	versionCmd.Flags().
		Bool("no-release-check", false, "do not check online for the latest release version")
}
