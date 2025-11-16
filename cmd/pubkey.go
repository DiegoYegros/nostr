package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	nostrkeys "nostr-cli/nostr"
)

var pubkeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Show your configured public key",
	Long:  "Read the config file and print the stored public key in both hex and npub format.",
	RunE:  runShowPublicKey,
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the configured public key",
	Long:  "Convenience alias for 'nostr pubkey' to display the stored public key.",
	RunE:  runShowPublicKey,
}

func init() {
	registerProfileFlag(pubkeyCmd)
	registerProfileFlag(whoamiCmd)
}

func runShowPublicKey(cmd *cobra.Command, args []string) error {
	_, profile, alias, err := loadProfileForCommand()
	if err != nil {
		return err
	}
	if profile.PublicKey == "" {
		return errors.New("no public key found; run 'nostr setup' first")
	}

	npub, err := nostrkeys.HexToNpub(profile.PublicKey)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] Public key (hex):  %s\n", alias, profile.PublicKey)
	fmt.Printf("[%s] Public key (npub): %s\n", alias, npub)
	return nil
}
