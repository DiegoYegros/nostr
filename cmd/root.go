package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"nostr-cli/nips/nip01"
	nostrkeys "nostr-cli/nostr"
)

var rootCmd = &cobra.Command{
	Use:   "nostr [message]",
	Short: "Nostr CLI",
	Long:  "A CLI client for Nostr.",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}

		sk, err := nostrkeys.PromptForDecryptedKey(cfg)
		if err != nil {
			return err
		}

		message := strings.Join(args, " ")
		ctx := context.Background()
		return nip01.PublishNote(ctx, cfg, sk, message)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(articleCmd)
}
