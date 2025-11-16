package cmd

import (
	"github.com/spf13/cobra"

	nostrkeys "nostr-cli/nostr"
)

var setupAlias string

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure your encrypted keys and relays",
	Long:  "Walk through an interactive prompt to generate or import your key, encrypt it, and set up relays.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nostrkeys.RunSetup(setupAlias)
	},
}

func init() {
	setupCmd.Flags().StringVar(&setupAlias, "alias", "default", "Profile alias to configure or update")
}
