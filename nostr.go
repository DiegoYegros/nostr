package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/nbd-wtf/go-nostr"
	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

type Config struct {
	Relays    []string `json:"relays"`
	PrivKey   string   `json:"encrypted_private_key"`
	Salt      string   `json:"salt"`
	PublicKey string   `json:"public_key"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "--setup":
		runSetup()
	case "--help":
		printUsage()
	case "article":
		runArticleCommand(os.Args[2:])
	default:
		runPublish(command)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  nostr --setup                  # Setup your Nostr private key")
	fmt.Println("  nostr --help                   # Show this help message")
	fmt.Println("  nostr article post <file>      # Publish a long-form article")
	fmt.Println("  nostr \"your message here\"      # Publish a note to Nostr")
}

func runArticleCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: nostr article post <file>")
		return
	}

	subcommand := args[0]

	switch subcommand {
	case "post":
		runArticlePost(args[1:])
	default:
		fmt.Printf("Unknown article command: %s\n", subcommand)
	}
}

func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(homeDir, ".config", "nostr", "config.json")
}

func loadConfig() (*Config, error) {
	configPath := getConfigPath()
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &config, nil
}

func decryptPrivateKeyFromConfig(config *Config) (string, error) {
	password, err := readPassword("Enter password to decrypt private key: ")
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}

	salt, err := hex.DecodeString(config.Salt)
	if err != nil {
		return "", fmt.Errorf("error decoding salt: %w", err)
	}

	sk, err := decrypt(config.PrivKey, password, salt)
	if err != nil {
		return "", fmt.Errorf("error decrypting private key: %w", err)
	}

	return sk, nil
}

func publishEventToRelays(relays []string, ev nostr.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, url := range relays {
		relay, err := nostr.RelayConnect(ctx, url)
		if err != nil {
			fmt.Printf("Failed to connect to %s: %v\n", url, err)
			continue
		}

		err = relay.Publish(ctx, ev)
		if err != nil {
			fmt.Printf("Failed to publish to %s: %v\n", url, err)
		} else {
			fmt.Printf("Published to %s\n", url)
		}
		relay.Close()
	}
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

func deriveKey(password string, salt []byte) ([]byte, error) {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32), nil
}

func encrypt(data []byte, password string, salt []byte) (string, error) {
	key, err := deriveKey(password, salt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(encryptedData string, password string, salt []byte) (string, error) {
	key, err := deriveKey(password, salt)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func nsecToHex(nsec string) (string, error) {
	hrp, data, err := bech32.Decode(nsec)
	if err != nil {
		return "", fmt.Errorf("invalid nsec format: %v", err)
	}

	if hrp != "nsec" {
		return "", fmt.Errorf("invalid prefix: expected nsec, got %s", hrp)
	}

	converted, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %v", err)
	}

	return hex.EncodeToString(converted), nil
}

func runSetup() {
	configPath := getConfigPath()
	os.MkdirAll(filepath.Dir(configPath), 0700)

	fmt.Print("Enter your private key (nsec format): ")
	var nsec string
	fmt.Scanln(&nsec)

	sk, err := nsecToHex(nsec)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		fmt.Println("Error: Invalid private key")
		return
	}

	password, err := readPassword("Enter password to encrypt private key: ")
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}

	confirmPassword, err := readPassword("Confirm password: ")
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}

	if password != confirmPassword {
		fmt.Println("Error: Passwords do not match")
		return
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		fmt.Println("Error generating salt:", err)
		return
	}

	encryptedKey, err := encrypt([]byte(sk), password, salt)
	if err != nil {
		fmt.Println("Error encrypting private key:", err)
		return
	}

	var config Config

	defaultRelays := []string{
		"wss://relay.damus.io",
		"wss://relay.primal.net",
		"wss://nos.lol",
	}

	if configData, err := os.ReadFile(configPath); err == nil {
		// Config exists, parse it
		if err := json.Unmarshal(configData, &config); err != nil {
			fmt.Println("Error parsing existing config:", err)
			return
		}
	} else {
		os.MkdirAll(filepath.Dir(configPath), 0700)
		config.Relays = defaultRelays
	}

	config.PrivKey = encryptedKey
	config.Salt = hex.EncodeToString(salt)
	config.PublicKey = pk

	configJson, _ := json.MarshalIndent(config, "", "  ")
	err = os.WriteFile(configPath, configJson, 0600)
	if err != nil {
		fmt.Println("Error saving config:", err)
		return
	}

	fmt.Println("Setup complete! Your public key is:", pk)
}

func runPublish(message string) {
	config, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	sk, err := decryptPrivateKeyFromConfig(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	ev := nostr.Event{
		PubKey:    config.PublicKey,
		CreatedAt: nostr.Now(),
		Kind:      1,
		Tags:      nil,
		Content:   message,
	}

	ev.Sign(sk)

	publishEventToRelays(config.Relays, ev)
}

func runArticlePost(args []string) {
	flagSet := flag.NewFlagSet("article post", flag.ContinueOnError)
	title := flagSet.String("title", "", "Title for the article")
	summary := flagSet.String("summary", "", "Short summary of the article")
	image := flagSet.String("image", "", "URL to an image representing the article")
	publishedAt := flagSet.String("published-at", "", "Published at timestamp")
	identifierFlag := flagSet.String("identifier", "", "Stable identifier for this article (used for the d tag)")
	flagSet.Usage = func() {
		fmt.Println("Usage: nostr article post [--title <title>] [--summary <summary>] [--image <url>] [--published-at <time>] [--identifier <id>] <file>")
	}

	if err := flagSet.Parse(args); err != nil {
		return
	}

	remaining := flagSet.Args()
	if len(remaining) != 1 {
		fmt.Println("Error: you must specify a file containing the article content.")
		flagSet.Usage()
		return
	}

	filePath := remaining[0]
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading article file: %v\n", err)
		return
	}

	articleBody := string(content)
	inferred := inferArticleMetadata(articleBody)

	config, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	sk, err := decryptPrivateKeyFromConfig(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	identifier := deriveArticleIdentifier(filePath, *identifierFlag)

	ev := nostr.Event{
		PubKey:    config.PublicKey,
		CreatedAt: nostr.Now(),
		Kind:      30023,
		Content:   articleBody,
		Tags:      nostr.Tags{{"d", identifier}},
	}

	effectiveTitle := strings.TrimSpace(*title)
	if effectiveTitle == "" {
		effectiveTitle = inferred.Title
	}
	if effectiveTitle == "" {
		base := filepath.Base(filePath)
		effectiveTitle = strings.TrimSpace(strings.TrimSuffix(base, filepath.Ext(base)))
	}
	if effectiveTitle == "" {
		effectiveTitle = identifier
	}
	if effectiveTitle != "" {
		ev.Tags = append(ev.Tags, nostr.Tag{"title", effectiveTitle})
	}

	effectiveSummary := strings.TrimSpace(*summary)
	if effectiveSummary == "" {
		effectiveSummary = inferred.Summary
	}
	if effectiveSummary != "" {
		ev.Tags = append(ev.Tags, nostr.Tag{"summary", effectiveSummary})
	}

	if *image != "" {
		ev.Tags = append(ev.Tags, nostr.Tag{"image", *image})
	}
	publishedAtValue := strings.TrimSpace(*publishedAt)
	if publishedAtValue == "" {
		publishedAtValue = fmt.Sprintf("%d", ev.CreatedAt)
	}
	ev.Tags = append(ev.Tags, nostr.Tag{"published_at", publishedAtValue})

	for _, relay := range config.Relays {
		relay = strings.TrimSpace(relay)
		if relay == "" {
			continue
		}
		ev.Tags = append(ev.Tags, nostr.Tag{"r", relay})
	}

	ev.Sign(sk)

	publishEventToRelays(config.Relays, ev)
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
