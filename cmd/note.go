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
	Use:   "note [message]",
	Short: "Publish a short note (NIP-01)",
	Long:  "Send a Kind 1 Nostr note to your configured relays using arguments or piped stdin.",
	RunE: func(cmd *cobra.Command, args []string) error {
		message := strings.TrimSpace(strings.Join(args, " "))
		if message == "" {
			input, ok, err := readInputFromStdin()
			if err != nil {
				return err
			}
			if ok {
				message = strings.TrimSpace(input)
			}
		}
		if message == "" {
			_ = cmd.Help()
			return fmt.Errorf("a note message is required")
		}

		_, profile, _, err := loadProfileForCommand()
		if err != nil {
			return err
		}

		sk, err := nostrkeys.PromptForDecryptedKey(profile)
		if err != nil {
			return err
		}

		ctx := context.Background()
		return nip01.PublishNote(ctx, profile, sk, message)
	},
}

func init() {
	registerProfileFlag(noteCmd)
}
