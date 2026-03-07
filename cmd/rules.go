package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage community rules",
	Long:  `Update or list the community-provided project cleanup rules.`,
}

var rulesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Fetch the latest community rules from GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("\n  🛰️  Fetching latest community rules...")

		url := "https://raw.githubusercontent.com/hariharen9/kessler/main/community-rules.yaml"
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to fetch rules: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch rules: server returned %s", resp.Status)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		configDir := filepath.Join(home, ".config", "kessler")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}

		destPath := filepath.Join(configDir, "community-rules.yaml")
		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}

		fmt.Println("  ✨ Done! Community rules updated at ~/.config/kessler/community-rules.yaml")
		fmt.Println("  Kessler will now use these rules alongside your local overrides.\n")
		return nil
	},
}

func init() {
	rulesCmd.AddCommand(rulesUpdateCmd)
	rootCmd.AddCommand(rulesCmd)
}
