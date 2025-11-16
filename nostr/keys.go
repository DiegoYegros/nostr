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
	"sort"
	"strings"
	"syscall"

	"github.com/btcsuite/btcd/btcutil/bech32"
	nostrlib "github.com/nbd-wtf/go-nostr"
	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

type Config struct {
	CurrentProfile string              `json:"current_profile"`
	Profiles       map[string]*Profile `json:"profiles"`
}

type Profile struct {
	Relays    []string `json:"relays"`
	PrivKey   string   `json:"encrypted_private_key"`
	Salt      string   `json:"salt"`
	PublicKey string   `json:"public_key"`
}

type legacyConfig struct {
	Relays    []string `json:"relays"`
	PrivKey   string   `json:"encrypted_private_key"`
	Salt      string   `json:"salt"`
	PublicKey string   `json:"public_key"`
}

func NewConfig() *Config {
	return &Config{Profiles: make(map[string]*Profile)}
}

func (cfg *Config) ensureProfiles() {
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
}

func (cfg *Config) ensureCurrentProfile() error {
	cfg.ensureProfiles()
	if len(cfg.Profiles) == 0 {
		return errors.New("no profiles configured; run 'nostr setup' first")
	}
	if cfg.CurrentProfile != "" {
		if _, ok := cfg.Profiles[cfg.CurrentProfile]; ok {
			return nil
		}
	}
	aliases := cfg.ProfileAliases()
	if len(aliases) == 0 {
		return errors.New("no profiles configured; run 'nostr setup' first")
	}
	cfg.CurrentProfile = aliases[0]
	return nil
}

func (cfg *Config) ProfileAliases() []string {
	cfg.ensureProfiles()
	var aliases []string
	for alias := range cfg.Profiles {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func (cfg *Config) ActiveProfile(aliasOverride string) (*Profile, string, error) {
	cfg.ensureProfiles()
	if len(cfg.Profiles) == 0 {
		return nil, "", errors.New("no profiles configured; run 'nostr setup' first")
	}
	target := strings.TrimSpace(aliasOverride)
	if target == "" {
		target = cfg.CurrentProfile
	}
	if target != "" {
		if profile, ok := cfg.Profiles[target]; ok {
			return profile, target, nil
		}
		return nil, "", fmt.Errorf("profile '%s' not found", target)
	}
	aliases := cfg.ProfileAliases()
	if len(aliases) == 0 {
		return nil, "", errors.New("no profiles configured; run 'nostr setup' first")
	}
	cfg.CurrentProfile = aliases[0]
	return cfg.Profiles[aliases[0]], cfg.CurrentProfile, nil
}

func (cfg *Config) SetCurrentProfile(alias string) error {
	cfg.ensureProfiles()
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return errors.New("profile alias cannot be empty")
	}
	if _, ok := cfg.Profiles[trimmed]; !ok {
		return fmt.Errorf("profile '%s' not found", trimmed)
	}
	cfg.CurrentProfile = trimmed
	return nil
}

func convertLegacyConfig(input legacyConfig) *Config {
	cfg := NewConfig()
	profile := &Profile{
		Relays:    append([]string{}, input.Relays...),
		PrivKey:   input.PrivKey,
		Salt:      input.Salt,
		PublicKey: input.PublicKey,
	}
	if len(profile.Relays) == 0 {
		profile.Relays = DefaultRelays()
	}
	cfg.Profiles["default"] = profile
	cfg.CurrentProfile = "default"
	return cfg
}

var defaultRelays = []string{
	"wss://relay.damus.io",
	"wss://relay.primal.net",
	"wss://nos.lol",
}

func DefaultRelays() []string {
	return append([]string{}, defaultRelays...)
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

	var probe map[string]json.RawMessage
	if err := json.Unmarshal(configData, &probe); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if _, ok := probe["profiles"]; ok {
		var cfg Config
		if err := json.Unmarshal(configData, &cfg); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
		if err := cfg.ensureCurrentProfile(); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	var legacy legacyConfig
	if err := json.Unmarshal(configData, &legacy); err != nil {
		return nil, fmt.Errorf("parsing legacy config: %w", err)
	}
	if legacy.PrivKey == "" || legacy.Salt == "" || legacy.PublicKey == "" {
		return nil, errors.New("config missing profile data; run 'nostr setup' again")
	}
	converted := convertLegacyConfig(legacy)
	if err := SaveConfig(converted); err != nil {
		return nil, err
	}
	return converted, nil
}

func SaveConfig(cfg *Config) error {
	configPath := GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return err
	}
	cfg.ensureProfiles()
	if len(cfg.Profiles) > 0 && cfg.CurrentProfile == "" {
		if err := cfg.ensureCurrentProfile(); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0o600)
}

func PromptForDecryptedKey(profile *Profile) (string, error) {
	password, err := readPassword("Enter password to decrypt private key: ")
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}

	salt, err := hex.DecodeString(profile.Salt)
	if err != nil {
		return "", fmt.Errorf("decoding salt: %w", err)
	}

	return decrypt(profile.PrivKey, password, salt)
}

func RunSetup(alias string) error {
	if strings.TrimSpace(alias) == "" {
		alias = "default"
	}
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

	return RunSetupWithKey(alias, sk)
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

func RunSetupWithKey(alias, sk string) error {
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

	if strings.TrimSpace(alias) == "" {
		alias = "default"
	}

	cfg, err := LoadConfig()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg = NewConfig()
	}

	cfg.ensureProfiles()
	relays := DefaultRelays()
	if existing, ok := cfg.Profiles[alias]; ok && len(existing.Relays) > 0 {
		relays = append([]string{}, existing.Relays...)
	}

	profile := &Profile{
		Relays:    relays,
		PrivKey:   encryptedKey,
		Salt:      hex.EncodeToString(salt),
		PublicKey: pk,
	}

	cfg.Profiles[alias] = profile
	cfg.CurrentProfile = alias

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Setup complete for '%s'! Your public key is: %s\n", alias, pk)
	return nil
}

func GenerateKeyPair() (string, string, error) {
	sk := nostrlib.GeneratePrivateKey()
	pk, err := nostrlib.GetPublicKey(sk)
	if err != nil {
		return "", "", err
	}
	return sk, pk, nil
}

func HexToNsec(sk string) (string, error) {
	return hexToBech32("nsec", sk)
}

func HexToNpub(pk string) (string, error) {
	return hexToBech32("npub", pk)
}

func hexToBech32(hrp, input string) (string, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}

	converted, err := bech32.ConvertBits(decoded, 8, 5, true)
	if err != nil {
		return "", err
	}

	return bech32.Encode(hrp, converted)
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
