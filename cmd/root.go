package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nostr",
	Short: "Nostr CLI toolkit",
	Long:  "A friendly CLI for managing your keys, relays, short notes, and NIP-23 articles.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
		fmt.Println()
		fmt.Println("Try \"nostr setup\" to configure your keys or \"nostr note \"hello\"\" to publish a message.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(articleCmd)
	rootCmd.AddCommand(relaysCmd)
	rootCmd.AddCommand(genKeysCmd)
	rootCmd.AddCommand(pubkeyCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(profileCmd)
}
