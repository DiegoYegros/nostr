package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	nostrkeys "nostr-cli/nostr"
)

var profileManagerCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage saved key profiles",
	Long:  "List available profile aliases, inspect the active one, and switch the default profile for future commands.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var profileListAliasesCmd = &cobra.Command{
	Use:   "list",
	Short: "Show available profile aliases",
	Long:  "Display every configured profile alias and highlight the current default.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Println("No profiles are configured. Run 'nostr setup --alias <name>' to create one.")
				return nil
			}
			return err
		}
		aliases := cfg.ProfileAliases()
		if len(aliases) == 0 {
			fmt.Println("No profiles are configured. Run 'nostr setup --alias <name>' to create one.")
			return nil
		}
		for _, alias := range aliases {
			marker := " "
			if alias == cfg.CurrentProfile {
				marker = "*"
			}
			fmt.Printf("%s %s\n", marker, alias)
		}
		return nil
	},
}

var profileAddCmd = &cobra.Command{
	Use:   "add <alias>",
	Short: "Create a new profile alias",
	Long:  "Walk through the encrypted key setup flow for a new alias so you can keep multiple profiles.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := strings.TrimSpace(args[0])
		if alias == "" {
			return fmt.Errorf("profile alias cannot be empty")
		}
		cfg, err := nostrkeys.LoadConfig()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if cfg != nil {
			if _, ok := cfg.Profiles[alias]; ok {
				return fmt.Errorf("profile '%s' already exists", alias)
			}
		}
		return nostrkeys.RunSetup(alias)
	},
}

var profileSwitchCmd = &cobra.Command{
	Use:   "switch <alias>",
	Short: "Switch the default profile",
	Long:  "Set the provided alias as the default profile used when --profile is not supplied.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])
		if target == "" {
			return fmt.Errorf("profile alias cannot be empty")
		}
		cfg, err := nostrkeys.LoadConfig()
		if err != nil {
			return err
		}
		if err := cfg.SetCurrentProfile(target); err != nil {
			return err
		}
		if err := nostrkeys.SaveConfig(cfg); err != nil {
			return err
		}
		fmt.Printf("Default profile set to '%s'\n", cfg.CurrentProfile)
		return nil
	},
}

func init() {
	profileManagerCmd.AddCommand(profileListAliasesCmd)
	profileManagerCmd.AddCommand(profileAddCmd)
	profileManagerCmd.AddCommand(profileSwitchCmd)
}
