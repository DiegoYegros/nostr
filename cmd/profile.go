package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"nostr-cli/nips/nip00"
	nostrkeys "nostr-cli/nostr"
)

var (
	profileName    string
	profileAbout   string
	profilePicture string
)

var profileCmd = &cobra.Command{
	Use:   "set-profile",
	Short: "Publish a Kind 0 profile event",
	Long:  "Send a metadata (Kind 0) event with your preferred name, about blurb, and picture URL.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profileName == "" && profileAbout == "" && profilePicture == "" {
			return errors.New("provide at least one of --name, --about, or --picture")
		}

		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}

		sk, err := nostrkeys.PromptForDecryptedKey(cfg)
		if err != nil {
			return err
		}

		metadata := nip00.ProfileMetadata{
			Name:    profileName,
			About:   profileAbout,
			Picture: profilePicture,
		}

		return nip00.PublishProfile(context.Background(), cfg, sk, metadata)
	},
}

func init() {
	profileCmd.Flags().StringVar(&profileName, "name", "", "Display name for your profile")
	profileCmd.Flags().StringVar(&profileAbout, "about", "", "Short bio or description")
	profileCmd.Flags().StringVar(&profilePicture, "picture", "", "Profile picture URL")
}
