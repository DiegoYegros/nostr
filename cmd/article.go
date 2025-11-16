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
	Use:   "article <file>",
	Short: "Publish a long-form article (NIP-23)",
	Long:  strings.TrimSpace(`Compose and publish a Markdown file as a NIP-23 article with optional metadata overrides.`),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			_ = cmd.Help()
			return fmt.Errorf("a path to the article Markdown file is required")
		}

		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}

		sk, err := nostrkeys.PromptForDecryptedKey(cfg)
		if err != nil {
			return err
		}

		opts := nip23.PublishOptions{
			FilePath:    args[0],
			Title:       articleTitle,
			Summary:     articleSummary,
			Image:       articleImage,
			PublishedAt: articlePublished,
			Identifier:  articleIdentifier,
		}

		return nip23.PublishArticle(context.Background(), cfg, sk, opts)
	},
}

func init() {
	articleCmd.Flags().StringVar(&articleTitle, "title", "", "Override the article title")
	articleCmd.Flags().StringVar(&articleSummary, "summary", "", "Override the summary blurb")
	articleCmd.Flags().StringVar(&articleImage, "image", "", "Set the preview image URL")
	articleCmd.Flags().StringVar(&articlePublished, "published-at", "", "Custom published-at timestamp")
	articleCmd.Flags().StringVar(&articleIdentifier, "identifier", "", "Stable identifier for the d tag")
}
