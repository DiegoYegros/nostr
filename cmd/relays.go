package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	nostrlib "github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"

	nostrkeys "nostr-cli/nostr"
)

var relaysCmd = &cobra.Command{
	Use:   "relays",
	Short: "List, edit, or synchronize your relay list",
	Long:  "Inspect or maintain the relay list stored in your config file, including pulling NIP-65 metadata.",
}

var relaysListCmd = &cobra.Command{
	Use:   "list",
	Short: "Print configured relays",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}
		if len(cfg.Relays) == 0 {
			fmt.Println("No relays are configured. Use 'nostr relays add <url>' to add one.")
			return nil
		}
		for i, relay := range cfg.Relays {
			fmt.Printf("%d. %s\n", i+1, strings.TrimSpace(relay))
		}
		return nil
	},
}

var relaysAddCmd = &cobra.Command{
	Use:   "add <relay> [relay...]",
	Short: "Add relay URLs to the config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			_ = cmd.Help()
			return fmt.Errorf("at least one relay URL is required")
		}
		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}
		added := addRelaysToConfig(cfg, args)
		if len(added) == 0 {
			fmt.Println("All provided relays are already configured.")
			return nil
		}
		if err := nostrkeys.SaveConfig(cfg); err != nil {
			return err
		}
		fmt.Printf("Added %d relay(s):\n", len(added))
		for _, relay := range added {
			fmt.Printf("- %s\n", relay)
		}
		return nil
	},
}

var relaysRemoveCmd = &cobra.Command{
	Use:   "remove <relay> [relay...]",
	Short: "Remove relay URLs from the config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			_ = cmd.Help()
			return fmt.Errorf("at least one relay URL is required")
		}
		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}
		removed, missing := removeRelaysFromConfig(cfg, args)
		if len(removed) == 0 {
			return fmt.Errorf("none of the provided relays were configured")
		}
		if err := nostrkeys.SaveConfig(cfg); err != nil {
			return err
		}
		fmt.Printf("Removed %d relay(s):\n", len(removed))
		for _, relay := range removed {
			fmt.Printf("- %s\n", relay)
		}
		if len(missing) > 0 {
			fmt.Printf("The following relays were not found: %s\n", strings.Join(missing, ", "))
		}
		return nil
	},
}

var relaysPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull relay metadata via the outbox model",
	Long:  "Connect to configured relays, fetch the latest kind 10002 event, and update the local relay list.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}
		pubKey := strings.TrimSpace(cfg.PublicKey)
		if pubKey == "" {
			return errors.New("no public key found in config; run 'nostr setup' first")
		}

		queryRelays := cfg.Relays
		if len(queryRelays) == 0 {
			queryRelays = nostrkeys.DefaultRelays()
		}

		ctx := context.Background()
		fetched, err := fetchRelaysFromOutbox(ctx, queryRelays, pubKey)
		if err != nil {
			return err
		}

		if len(fetched) == 0 {
			return errors.New("no writable relays were advertised by your outbox")
		}

		cfg.Relays = fetched
		if err := nostrkeys.SaveConfig(cfg); err != nil {
			return err
		}
		fmt.Printf("Synchronized %d relay(s) from outbox metadata:\n", len(fetched))
		for _, relay := range fetched {
			fmt.Printf("- %s\n", relay)
		}
		return nil
	},
}

func init() {
	relaysCmd.AddCommand(relaysListCmd)
	relaysCmd.AddCommand(relaysAddCmd)
	relaysCmd.AddCommand(relaysRemoveCmd)
	relaysCmd.AddCommand(relaysPullCmd)
}

func addRelaysToConfig(cfg *nostrkeys.Config, relays []string) []string {
	existing := make(map[string]struct{})
	for _, relay := range cfg.Relays {
		key := normalizedRelayKey(relay)
		if key == "" {
			continue
		}
		existing[key] = struct{}{}
	}

	var added []string
	for _, relay := range relays {
		trimmed := cleanRelayURL(relay)
		if trimmed == "" {
			continue
		}
		key := normalizedRelayKey(trimmed)
		if key == "" {
			continue
		}
		if _, ok := existing[key]; ok {
			continue
		}
		existing[key] = struct{}{}
		cfg.Relays = append(cfg.Relays, trimmed)
		added = append(added, trimmed)
	}
	return added
}

func removeRelaysFromConfig(cfg *nostrkeys.Config, targets []string) (removed []string, missing []string) {
	targetMap := make(map[string]string)
	for _, relay := range targets {
		trimmed := cleanRelayURL(relay)
		if trimmed == "" {
			continue
		}
		key := normalizedRelayKey(trimmed)
		if key == "" {
			continue
		}
		targetMap[key] = relay
	}

	var remaining []string
	for _, relay := range cfg.Relays {
		key := normalizedRelayKey(relay)
		if _, ok := targetMap[key]; ok {
			removed = append(removed, relay)
			delete(targetMap, key)
			continue
		}
		remaining = append(remaining, relay)
	}
	cfg.Relays = remaining

	for _, relay := range targetMap {
		missing = append(missing, relay)
	}
	return removed, missing
}

func fetchRelaysFromOutbox(ctx context.Context, candidateRelays []string, pubKey string) ([]string, error) {
	seen := make(map[string]struct{})
	var discovered []string

	for _, url := range candidateRelays {
		trimmed := cleanRelayURL(url)
		if trimmed == "" {
			continue
		}

		connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		relay, err := nostrlib.RelayConnect(connectCtx, trimmed)
		cancel()
		if err != nil {
			fmt.Printf("Failed to connect to %s: %v\n", trimmed, err)
			continue
		}

		queryCtx, queryCancel := context.WithTimeout(ctx, 5*time.Second)
		events, err := relay.QuerySync(queryCtx, nostrlib.Filter{Authors: []string{pubKey}, Kinds: []int{10002}, Limit: 1})
		queryCancel()
		relay.Close()
		if err != nil {
			fmt.Printf("Failed to query %s: %v\n", trimmed, err)
			continue
		}
		if len(events) == 0 {
			continue
		}

		for _, ev := range events {
			for _, tag := range ev.Tags {
				if len(tag) < 2 || tag[0] != "r" {
					continue
				}
				permission := ""
				if len(tag) >= 3 {
					permission = strings.ToLower(strings.TrimSpace(tag[2]))
				}
				if permission == "read" {
					continue
				}
				relayURL := cleanRelayURL(tag[1])
				if relayURL == "" {
					continue
				}
				key := normalizedRelayKey(relayURL)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				discovered = append(discovered, relayURL)
			}
		}
	}

	if len(discovered) == 0 {
		return nil, errors.New("no relay list metadata was found on the queried relays")
	}

	return discovered, nil
}

func cleanRelayURL(url string) string {
	trimmed := strings.TrimSpace(url)
	trimmed = strings.TrimRight(trimmed, "/")
	return trimmed
}

func normalizedRelayKey(url string) string {
	trimmed := cleanRelayURL(url)
	return strings.ToLower(trimmed)
}
