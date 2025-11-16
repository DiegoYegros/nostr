package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	nostrkeys "nostr-cli/nostr"
)

var genKeysAlias string

var genKeysCmd = &cobra.Command{
	Use:   "gen-keys",
	Short: "Generate a new private/public key pair",
	Long:  "Create a brand new private/public key pair and continue into the encrypted setup flow.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sk, pk, err := nostrkeys.GenerateKeyPair()
		if err != nil {
			return err
		}

		nsec, err := nostrkeys.HexToNsec(sk)
		if err != nil {
			return err
		}

		npub, err := nostrkeys.HexToNpub(pk)
		if err != nil {
			return err
		}

		fmt.Println("Generated keys:")
		fmt.Printf("  Secret (hex):  %s\n", sk)
		fmt.Printf("  Secret (nsec): %s\n", nsec)
		fmt.Printf("  Public (hex):  %s\n", pk)
		fmt.Printf("  Public (npub): %s\n", npub)
		fmt.Println()
		fmt.Println("Continuing with setup to encrypt and save your new keys...")
		return nostrkeys.RunSetupWithKey(genKeysAlias, sk)
	},
}

func init() {
	genKeysCmd.Flags().StringVar(&genKeysAlias, "alias", "default", "Profile alias to store the generated key")
}
