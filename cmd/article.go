package cmd

import (
	"context"

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
	Use:   "article",
	Short: "NIP-23 long-form article utilities",
}

var articlePostCmd = &cobra.Command{
	Use:   "post <file>",
	Short: "Publish a long-form article (NIP-23)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
	articleCmd.AddCommand(articlePostCmd)

	articlePostCmd.Flags().StringVar(&articleTitle, "title", "", "Article title")
	articlePostCmd.Flags().StringVar(&articleSummary, "summary", "", "Article summary")
	articlePostCmd.Flags().StringVar(&articleImage, "image", "", "Image URL")
	articlePostCmd.Flags().StringVar(&articlePublished, "published-at", "", "Published-at timestamp")
	articlePostCmd.Flags().StringVar(&articleIdentifier, "identifier", "", "Stable identifier for the d tag")
}
