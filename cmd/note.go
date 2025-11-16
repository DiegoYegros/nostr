package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"nostr-cli/nips/nip01"
	nostrkeys "nostr-cli/nostr"
)

var noteCmd = &cobra.Command{
	Use:   "note <message>",
	Short: "Publish a short note (NIP-01)",
	Long:  "Send a Kind 1 Nostr note to every relay configured in your config.json file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			_ = cmd.Help()
			return fmt.Errorf("a note message is required")
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
