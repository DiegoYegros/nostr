package nostr

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/btcsuite/btcd/btcutil/bech32"
	nostrlib "github.com/nbd-wtf/go-nostr"
	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

type Config struct {
	Relays    []string `json:"relays"`
	PrivKey   string   `json:"encrypted_private_key"`
	Salt      string   `json:"salt"`
	PublicKey string   `json:"public_key"`
}

var defaultRelays = []string{
	"wss://relay.damus.io",
	"wss://relay.primal.net",
	"wss://nos.lol",
}

func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(homeDir, ".config", "nostr", "config.json")
}

func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &config, nil
}

func SaveConfig(cfg *Config) error {
	configPath := GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0o600)
}

func PromptForDecryptedKey(cfg *Config) (string, error) {
	password, err := readPassword("Enter password to decrypt private key: ")
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}

	salt, err := hex.DecodeString(cfg.Salt)
	if err != nil {
		return "", fmt.Errorf("decoding salt: %w", err)
	}

	return decrypt(cfg.PrivKey, password, salt)
}

func RunSetup() error {
	configPath := GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return err
	}

	fmt.Print("Enter your private key (nsec format): ")
	var nsec string
	if _, err := fmt.Scanln(&nsec); err != nil {
		return err
	}

	sk, err := nsecToHex(strings.TrimSpace(nsec))
	if err != nil {
		return err
	}

	pk, err := nostrlib.GetPublicKey(sk)
	if err != nil {
		return errors.New("invalid private key provided")
	}

	password, err := readPassword("Enter password to encrypt private key: ")
	if err != nil {
		return err
	}
	confirm, err := readPassword("Confirm password: ")
	if err != nil {
		return err
	}
	if password != confirm {
		return errors.New("passwords do not match")
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}

	encryptedKey, err := encrypt([]byte(sk), password, salt)
	if err != nil {
		return err
	}

	cfg := &Config{
		Relays:    defaultRelays,
		PrivKey:   encryptedKey,
		Salt:      hex.EncodeToString(salt),
		PublicKey: pk,
	}

	if existing, err := LoadConfig(); err == nil {
		cfg.Relays = existing.Relays
	}

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Println("Setup complete! Your public key is:", pk)
	return nil
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

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
}

func encrypt(data []byte, password string, salt []byte) (string, error) {
	key := deriveKey(password, salt)
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
	key := deriveKey(password, salt)
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
