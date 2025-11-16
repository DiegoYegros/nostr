# Nostr on the CLI.

Have all your configs in a single config.json.

This includes:
1. Relays
2. Encrypted Private Key
3. Salt

Use `nostr setup` to introduce your key. You will be asked a password to decrypt it.

Use `nostr note "This is a note of Kind 1"` to send the note to your relays.

Use `nostr article path/to/article.md` to publish a long-form NIP-23 article. Flags such as `--title`, `--summary`, `--image`, `--published-at`, and `--identifier` are available for metadata overrides.

## Supported NIPs:
- NIP-10 Text Notes
- NIP-23 Long Form Content 
