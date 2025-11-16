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

func runShowPublicKey(cmd *cobra.Command, args []string) error {
	cfg, err := nostrkeys.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.PublicKey == "" {
		return errors.New("no public key found; run 'nostr setup' first")
	}

	npub, err := nostrkeys.HexToNpub(cfg.PublicKey)
	if err != nil {
		return err
	}

	fmt.Printf("Public key (hex):  %s\n", cfg.PublicKey)
	fmt.Printf("Public key (npub): %s\n", npub)
	return nil
}
