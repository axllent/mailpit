package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// MailpitLicense is populated from the embedded LICENSE file in main.go
	MailpitLicense string

	// ThirdPartyLicenses is populated from the embedded third-party license file
	ThirdPartyLicenses string
)

var licensesCmd = &cobra.Command{
	Use:     "licenses",
	Short:   "Display software license information",
	Long:    "Display the Mailpit or third-party dependency license information.",
	Aliases: []string{"licences"},
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Help()
	},
}

var licenseMailpitCmd = &cobra.Command{
	Use:   "mailpit",
	Short: "Display the Mailpit license (MIT)",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Print(MailpitLicense)
	},
}

var licenseThirdPartyCmd = &cobra.Command{
	Use:   "third-party",
	Short: "Display third-party dependency licenses",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Print(ThirdPartyLicenses)
	},
}

func init() {
	rootCmd.AddCommand(licensesCmd)
	licensesCmd.AddCommand(licenseMailpitCmd)
	licensesCmd.AddCommand(licenseThirdPartyCmd)
}
