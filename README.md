# Nostr on the CLI
<img width="1382" height="513" alt="image" src="https://github.com/user-attachments/assets/2d452362-95e8-42a3-814b-1934244fed24" />

Have all your configs in a single config.json.

This includes:
1. Relays
2. Encrypted Private Key
3. Salt

Use `nostr setup` to introduce your key. You will be asked a password to decrypt it.

Use `nostr note "This is a note of Kind 1"` to send the note to your relays.

Use `nostr relays list` to inspect the relays stored in your config, `nostr relays add <url>` or `nostr relays remove <url>` to edit the list.

Use `nostr article path/to/article.md` to publish a long-form NIP-23 article. Flags such as `--title`, `--summary`, `--image`, `--published-at`, and `--identifier` are available for metadata overrides.

## Supported NIPs
- NIP-01 Text Notes
- NIP-23 Long Form Content 
