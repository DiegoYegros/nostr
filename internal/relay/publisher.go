package relay

import (
	"context"
	"fmt"
	"time"

	nostrlib "github.com/nbd-wtf/go-nostr"
)

func PublishToRelays(ctx context.Context, relays []string, ev nostrlib.Event) {
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, url := range relays {
		relay, err := nostrlib.RelayConnect(publishCtx, url)
		if err != nil {
			fmt.Printf("Failed to connect to %s: %v\n", url, err)
			continue
		}

		if err := relay.Publish(publishCtx, ev); err != nil {
			fmt.Printf("Failed to publish to %s: %v\n", url, err)
		} else {
			fmt.Printf("Published to %s\n", url)
		}
		relay.Close()
	}
}
