package nip23

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	nostrlib "github.com/nbd-wtf/go-nostr"

	"nostr-cli/internal/relay"
	nostrkeys "nostr-cli/nostr"
)

type PublishOptions struct {
	FilePath    string
	Title       string
	Summary     string
	Image       string
	PublishedAt string
	Identifier  string
}

func PublishArticle(ctx context.Context, cfg *nostrkeys.Config, sk string, opts PublishOptions) error {
	content, err := os.ReadFile(opts.FilePath)
	if err != nil {
		return fmt.Errorf("reading article file: %w", err)
	}

	body := string(content)
	inferred := inferArticleMetadata(body)
	identifier := deriveArticleIdentifier(opts.FilePath, opts.Identifier)

	ev := nostrlib.Event{
		PubKey:    cfg.PublicKey,
		CreatedAt: nostrlib.Now(),
		Kind:      30023,
		Content:   body,
		Tags:      nostrlib.Tags{{"d", identifier}},
	}

	effectiveTitle := fallbackValue(opts.Title, inferred.Title, identifierFromPath(opts.FilePath), identifier)
	if effectiveTitle != "" {
		ev.Tags = append(ev.Tags, nostrlib.Tag{"title", effectiveTitle})
	}

	effectiveSummary := fallbackValue(opts.Summary, inferred.Summary)
	if effectiveSummary != "" {
		ev.Tags = append(ev.Tags, nostrlib.Tag{"summary", effectiveSummary})
	}

	if opts.Image != "" {
		ev.Tags = append(ev.Tags, nostrlib.Tag{"image", opts.Image})
	}

	publishedAt := strings.TrimSpace(opts.PublishedAt)
	if publishedAt == "" {
		publishedAt = fmt.Sprintf("%d", ev.CreatedAt)
	}
	ev.Tags = append(ev.Tags, nostrlib.Tag{"published_at", publishedAt})

	for _, relayURL := range cfg.Relays {
		relayURL = strings.TrimSpace(relayURL)
		if relayURL == "" {
			continue
		}
		ev.Tags = append(ev.Tags, nostrlib.Tag{"r", relayURL})
	}

	if err := ev.Sign(sk); err != nil {
		return err
	}

	relay.PublishToRelays(ctx, cfg.Relays, ev)
	return nil
}

func fallbackValue(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type articleMetadata struct {
	Title   string
	Summary string
}

func inferArticleMetadata(content string) articleMetadata {
	lines := strings.Split(content, "\n")
	metadata := articleMetadata{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			metadata.Title = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			break
		}
	}

	var summaryLines []string
	collecting := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if collecting {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if collecting {
				break
			}
			continue
		}
		summaryLines = append(summaryLines, trimmed)
		collecting = true
	}

	metadata.Summary = strings.Join(summaryLines, " ")
	return metadata
}

func deriveArticleIdentifier(filePath, provided string) string {
	if provided != "" {
		return provided
	}

	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	slug := slugifyIdentifier(name)
	if slug == "" {
		slug = slugifyIdentifier(base)
	}
	if slug == "" {
		hash := sha256.Sum256([]byte(filePath))
		return fmt.Sprintf("article-%x", hash[:4])
	}

	return slug
}

func identifierFromPath(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSpace(strings.TrimSuffix(base, filepath.Ext(base)))
}

func slugifyIdentifier(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	lastHyphen := false
	for _, r := range input {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '_' || r == '-':
			if !lastHyphen && b.Len() > 0 {
				b.WriteRune('-')
				lastHyphen = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}
