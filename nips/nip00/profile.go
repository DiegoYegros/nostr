package nip00

import (
	"context"
	"encoding/json"

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
