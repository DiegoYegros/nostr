package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"nostr-cli/nips/nip00"
	nostrkeys "nostr-cli/nostr"
)

var (
	profileName      string
	profileAbout     string
	profilePicture   string
	getProfilePubKey string
)

var profileCmd = &cobra.Command{
	Use:   "set-profile",
	Short: "Publish a Kind 0 profile event",
	Long:  "Send a metadata (Kind 0) event with your preferred name, about blurb, and picture URL.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profileName == "" && profileAbout == "" && profilePicture == "" {
			return errors.New("provide at least one of --name, --about, or --picture")
		}

		_, activeProfile, _, err := loadProfileForCommand()
		if err != nil {
			return err
		}

		sk, err := nostrkeys.PromptForDecryptedKey(activeProfile)
		if err != nil {
			return err
		}

		metadata := nip00.ProfileMetadata{
			Name:    profileName,
			About:   profileAbout,
			Picture: profilePicture,
		}

		if profileName == "" || profileAbout == "" || profilePicture == "" {
			existing, err := nip00.FetchProfile(context.Background(), activeProfile.Relays, activeProfile.PublicKey)
			if err == nil && existing != nil {
				if metadata.Name == "" {
					metadata.Name = existing.Name
				}
				if metadata.About == "" {
					metadata.About = existing.About
				}
				if metadata.Picture == "" {
					metadata.Picture = existing.Picture
				}
			}
		}

		return nip00.PublishProfile(context.Background(), activeProfile, sk, metadata)
	},
}

var getProfileCmd = &cobra.Command{
	Use:   "get-profile",
	Short: "Show Kind 0 profile metadata",
	Long:  "Fetch the latest metadata (Kind 0) event for your account or another public key from your configured relays.",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, activeProfile, _, err := loadProfileForCommand()
		if err != nil {
			return err
		}

		pubKey := getProfilePubKey
		if pubKey == "" {
			pubKey = activeProfile.PublicKey
		}

		profile, err := nip00.FetchProfile(context.Background(), activeProfile.Relays, pubKey)
		if err != nil {
			return err
		}

		if profile == nil {
			fmt.Println("No profile metadata found.")
			return nil
		}

		output, err := json.MarshalIndent(profile, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(output))
		return nil
	},
}

func init() {
	profileCmd.Flags().StringVar(&profileName, "name", "", "Display name for your profile")
	profileCmd.Flags().StringVar(&profileAbout, "about", "", "Short bio or description")
	profileCmd.Flags().StringVar(&profilePicture, "picture", "", "Profile picture URL")
	getProfileCmd.Flags().StringVar(&getProfilePubKey, "pubkey", "", "Hex public key to inspect (defaults to your configured key)")
	registerProfileFlag(profileCmd)
	registerProfileFlag(getProfileCmd)
}
