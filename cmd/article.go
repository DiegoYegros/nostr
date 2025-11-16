package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"nostr-cli/nips/nip23"
	nostrkeys "nostr-cli/nostr"
)

var (
	articleTitle      string
	articleSummary    string
	articleImage      string
	articlePublished  string
	articleIdentifier string
)

var articleCmd = &cobra.Command{
	Use:   "article [file]",
	Short: "Publish a long-form article (NIP-23)",
	Long:  strings.TrimSpace(`Compose and publish Markdown as a NIP-23 article from a file or piped stdin with optional metadata overrides.`),
	RunE: func(cmd *cobra.Command, args []string) error {
		var filePath string
		if len(args) > 0 && args[0] != "-" {
			filePath = args[0]
		}

		var inline string
		if filePath == "" {
			input, ok, err := readInputFromStdin()
			if err != nil {
				return err
			}
			if !ok {
				_ = cmd.Help()
				return fmt.Errorf("provide a Markdown file path or pipe content into the command")
			}
			inline = input
		}

		_, profile, _, err := loadProfileForCommand()
		if err != nil {
			return err
		}

		sk, err := nostrkeys.PromptForDecryptedKey(profile)
		if err != nil {
			return err
		}

		opts := nip23.PublishOptions{
			FilePath:      filePath,
			InlineContent: inline,
			Title:         articleTitle,
			Summary:       articleSummary,
			Image:         articleImage,
			PublishedAt:   articlePublished,
			Identifier:    articleIdentifier,
		}

		return nip23.PublishArticle(context.Background(), profile, sk, opts)
	},
}

func init() {
	articleCmd.Flags().StringVar(&articleTitle, "title", "", "Override the article title")
	articleCmd.Flags().StringVar(&articleSummary, "summary", "", "Override the summary blurb")
	articleCmd.Flags().StringVar(&articleImage, "image", "", "Set the preview image URL")
	articleCmd.Flags().StringVar(&articlePublished, "published-at", "", "Custom published-at timestamp")
	articleCmd.Flags().StringVar(&articleIdentifier, "identifier", "", "Stable identifier for the d tag")
	registerProfileFlag(articleCmd)
}
