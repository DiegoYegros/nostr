package cmd

import (
	"github.com/spf13/cobra"

	nostrkeys "nostr-cli/nostr"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure your encrypted keys and relays",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nostrkeys.RunSetup()
	},
}
