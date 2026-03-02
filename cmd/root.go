package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "shadiff",
	Short: "Shadow traffic semantic diff tool",
	Long: `Shadiff - Shadow traffic semantic diff tool

Validates behavioral consistency of cross-framework/cross-language API migrations
through a black-box record-replay-diff three-stage workflow.

Workflow:
  1. shadiff record  — Record the old API behavior (requests/responses/DB side effects)
  2. shadiff replay  — Replay recorded traffic to the new API
  3. shadiff diff    — Perform semantic-level comparison of behavioral differences
  4. shadiff report  — Generate a detailed diff report`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default ~/.shadiff/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose logs")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Show errors only")

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	// Set version info
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
