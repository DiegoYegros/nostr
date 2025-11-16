package cmd

import (
	"github.com/spf13/cobra"

	nostrkeys "nostr-cli/nostr"
)

func loadProfileForCommand() (*nostrkeys.Config, *nostrkeys.Profile, string, error) {
	cfg, err := nostrkeys.LoadConfig()
	if err != nil {
		return nil, nil, "", err
	}
	profile, alias, err := cfg.ActiveProfile(profileOverride)
	if err != nil {
		return nil, nil, "", err
	}
	return cfg, profile, alias, nil
}

func registerProfileFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&profileOverride, "profile", "", "Use the named profile for this command")
}
