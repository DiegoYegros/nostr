package nip00

import (
	"context"
	"encoding/json"
	"errors"

	nostrlib "github.com/nbd-wtf/go-nostr"

	"nostr-cli/internal/relay"
	nostrkeys "nostr-cli/nostr"
)

type ProfileMetadata struct {
	Name    string `json:"name,omitempty"`
	About   string `json:"about,omitempty"`
	Picture string `json:"picture,omitempty"`
}

func PublishProfile(ctx context.Context, cfg *nostrkeys.Config, sk string, profile ProfileMetadata) error {
	content, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	ev := nostrlib.Event{
		PubKey:    cfg.PublicKey,
		CreatedAt: nostrlib.Now(),
		Kind:      0,
		Content:   string(content),
	}

	if err := ev.Sign(sk); err != nil {
		return err
	}

	relay.PublishToRelays(ctx, cfg.Relays, ev)
	return nil
}

func FetchProfile(ctx context.Context, relays []string, pubKey string) (*ProfileMetadata, error) {
	if pubKey == "" {
		return nil, errors.New("a public key is required")
	}

	for _, url := range relays {
		relay, err := nostrlib.RelayConnect(ctx, url)
		if err != nil {
			continue
		}

		events, err := relay.QuerySync(ctx, nostrlib.Filter{Kinds: []int{0}, Authors: []string{pubKey}, Limit: 1})
		relay.Close()
		if err != nil || len(events) == 0 {
			continue
		}

		var profile ProfileMetadata
		if err := json.Unmarshal([]byte(events[0].Content), &profile); err != nil {
			continue
		}

		return &profile, nil
	}

	return nil, errors.New("profile not found on configured relays")
}
