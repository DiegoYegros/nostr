package nip01

import (
	"context"

	nostrlib "github.com/nbd-wtf/go-nostr"

	"nostr-cli/internal/relay"
	nostrkeys "nostr-cli/nostr"
)

func PublishNote(ctx context.Context, profile *nostrkeys.Profile, sk, message string) error {
	ev := nostrlib.Event{
		PubKey:    profile.PublicKey,
		CreatedAt: nostrlib.Now(),
		Kind:      1,
		Content:   message,
	}

	if err := ev.Sign(sk); err != nil {
		return err
	}

	relay.PublishToRelays(ctx, profile.Relays, ev)
	return nil
}
