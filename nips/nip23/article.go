package nip23

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	nostrlib "github.com/nbd-wtf/go-nostr"

	"nostr-cli/internal/relay"
	nostrkeys "nostr-cli/nostr"
)

type PublishOptions struct {
	FilePath      string
	InlineContent string
	Title         string
	Summary       string
	Image         string
	PublishedAt   string
	Identifier    string
}

func PublishArticle(ctx context.Context, profile *nostrkeys.Profile, sk string, opts PublishOptions) error {
	var body string
	switch {
	case strings.TrimSpace(opts.FilePath) != "":
		content, err := os.ReadFile(opts.FilePath)
		if err != nil {
			return fmt.Errorf("reading article file: %w", err)
		}
		body = strings.TrimPrefix(string(content), "\ufeff")
	case opts.InlineContent != "":
		body = strings.TrimPrefix(opts.InlineContent, "\ufeff")
	default:
		return fmt.Errorf("article content is required")
	}
	frontMatter, strippedBody := extractFrontMatter(body)
	body = strippedBody
	inferred := inferArticleMetadata(body)
	identifier := deriveArticleIdentifier(opts.FilePath, fallbackValue(opts.Identifier, frontMatter.Identifier), body)

	ev := nostrlib.Event{
		PubKey:    profile.PublicKey,
		CreatedAt: nostrlib.Now(),
		Kind:      30023,
		Content:   body,
		Tags:      nostrlib.Tags{{"d", identifier}},
	}

	effectiveTitle := fallbackValue(opts.Title, frontMatter.Title, inferred.Title, identifierFromPath(opts.FilePath), identifier)
	if effectiveTitle != "" {
		ev.Tags = append(ev.Tags, nostrlib.Tag{"title", effectiveTitle})
	}

	effectiveSummary := fallbackValue(opts.Summary, frontMatter.Summary, inferred.Summary)
	if effectiveSummary != "" {
		ev.Tags = append(ev.Tags, nostrlib.Tag{"summary", effectiveSummary})
	}

	effectiveImage := fallbackValue(opts.Image, frontMatter.Image)
	if effectiveImage != "" {
		ev.Tags = append(ev.Tags, nostrlib.Tag{"image", effectiveImage})
	}

	publishedAt := strings.TrimSpace(opts.PublishedAt)
	if publishedAt == "" {
		publishedAt = strings.TrimSpace(frontMatter.PublishedAt)
	}
	if publishedAt == "" {
		if normalized, ok := normalizeFrontMatterDate(frontMatter.Date); ok {
			publishedAt = normalized
		}
	}
	if publishedAt == "" {
		publishedAt = fmt.Sprintf("%d", ev.CreatedAt)
	}
	ev.Tags = append(ev.Tags, nostrlib.Tag{"published_at", publishedAt})

	for _, relayURL := range profile.Relays {
		relayURL = strings.TrimSpace(relayURL)
		if relayURL == "" {
			continue
		}
		ev.Tags = append(ev.Tags, nostrlib.Tag{"r", relayURL})
	}

	if err := ev.Sign(sk); err != nil {
		return err
	}

	relay.PublishToRelays(ctx, profile.Relays, ev)
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
	Title       string
	Summary     string
	Image       string
	PublishedAt string
	Date        string
	Identifier  string
}

func extractFrontMatter(content string) (articleMetadata, string) {
	metadata := articleMetadata{}
	lines := strings.Split(content, "\n")
	firstMarker := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "---" {
			firstMarker = i
		}
		break
	}
	if firstMarker == -1 {
		return metadata, content
	}
	secondMarker := -1
	for i := firstMarker + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			secondMarker = i
			break
		}
	}
	if secondMarker == -1 {
		return metadata, content
	}
	block := strings.Join(lines[firstMarker+1:secondMarker], "\n")
	metadata = parseFrontMatterBlock(block)
	remaining := append([]string{}, lines[:firstMarker]...)
	remaining = append(remaining, lines[secondMarker+1:]...)
	cleaned := strings.Join(remaining, "\n")
	cleaned = strings.TrimLeft(cleaned, "\r\n")
	return metadata, cleaned
}

func parseFrontMatterBlock(block string) articleMetadata {
	metadata := articleMetadata{}
	lines := strings.Split(block, "\n")
	var currentKey string
	collecting := false
	var buffer []string

	flush := func() {
		if !collecting {
			return
		}
		value := strings.Join(buffer, "\n")
		value = strings.TrimSpace(value)
		applyMetadataValue(&metadata, currentKey, value)
		currentKey = ""
		buffer = nil
		collecting = false
	}

	for i := 0; i < len(lines); {
		line := strings.TrimRight(lines[i], "\r")
		trimmed := strings.TrimSpace(line)
		if collecting {
			if trimmed == "" {
				buffer = append(buffer, "")
				i++
				continue
			}
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				buffer = append(buffer, strings.TrimLeft(line, " \t"))
				i++
				continue
			}
			flush()
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			i++
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if value == "|" || value == ">" {
			collecting = true
			currentKey = key
			buffer = buffer[:0]
			i++
			continue
		}
		applyMetadataValue(&metadata, key, trimQuotedValue(value))
		i++
	}
	if collecting {
		flush()
	}
	return metadata
}

func trimQuotedValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func applyMetadataValue(metadata *articleMetadata, key, value string) {
	switch key {
	case "title":
		metadata.Title = value
	case "summary":
		metadata.Summary = value
	case "image":
		metadata.Image = value
	case "published-at":
		metadata.PublishedAt = value
	case "date":
		metadata.Date = value
	case "identifier":
		metadata.Identifier = value
	}
}

func normalizeFrontMatterDate(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 -07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return fmt.Sprintf("%d", t.Unix()), true
		}
	}
	if unix, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return fmt.Sprintf("%d", unix), true
	}
	return "", false
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

func deriveArticleIdentifier(filePath, provided, content string) string {
	if provided != "" {
		return provided
	}

	if strings.TrimSpace(filePath) == "" {
		trimmed := strings.TrimSpace(content)
		if trimmed != "" {
			hash := sha256.Sum256([]byte(trimmed))
			return fmt.Sprintf("article-%x", hash[:4])
		}
		hash := sha256.Sum256([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
		return fmt.Sprintf("article-%x", hash[:4])
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
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" || trimmed == "." {
		return ""
	}
	base := filepath.Base(trimmed)
	if base == "" || base == "." {
		return ""
	}
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
