package cmd

import (
	"fmt"
	"os"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/replay"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var (
	replaySession     string
	replayTarget      string
	replayConcurrency int
	replayDelay       string
	replayDBProxy     []string
)

var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Replay recorded traffic",
	Long: `Read requests from a recorded session, send them sequentially to the target service, and record new responses.

Examples:
  shadiff replay -s abc123 -t http://new-api:9090
  shadiff replay -s "user-module-migration" -t http://localhost:9090 -c 5`,
	RunE: runReplay,
}

func init() {
	replayCmd.Flags().StringVarP(&replaySession, "session", "s", "", "Session ID or name (required)")
	replayCmd.Flags().StringVarP(&replayTarget, "target", "t", "", "Replay target address (required, e.g. http://localhost:9090)")
	replayCmd.Flags().IntVarP(&replayConcurrency, "concurrency", "c", 1, "Concurrency level")
	replayCmd.Flags().StringVar(&replayDelay, "delay", "", "Delay between requests (e.g. 100ms)")
	replayCmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "DB proxy (e.g. mysql://:13307->:3306)")

	replayCmd.MarkFlagRequired("session")
	replayCmd.MarkFlagRequired("target")
	rootCmd.AddCommand(replayCmd)
}

func runReplay(cmd *cobra.Command, args []string) error {
	// Initialize logger
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/.shadiff"
	if err := logger.Init(dataDir); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()

	// Create storage
	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Find session
	sessionID, err := resolveSession(store, replaySession)
	if err != nil {
		return err
	}

	// Parse delay
	var delay time.Duration
	if replayDelay != "" {
		delay, err = time.ParseDuration(replayDelay)
		if err != nil {
			return fmt.Errorf("invalid delay: %w", err)
		}
	}

	// Create replay engine
	engine := replay.NewEngine(store, replay.EngineConfig{
		SessionID:   sessionID,
		TargetURL:   replayTarget,
		Concurrency: replayConcurrency,
		Delay:       delay,
	})

	// Execute replay
	results, err := engine.Run()
	if err != nil {
		return fmt.Errorf("replay failed: %w", err)
	}

	// Update session status
	session, err := store.Get(sessionID)
	if err == nil {
		session.Status = model.SessionReplayed
		session.Target = model.EndpointConfig{BaseURL: replayTarget}
		store.Update(session)
	}

	// Print summary
	errorCount := 0
	for _, r := range results {
		if r.Error != nil {
			errorCount++
		}
	}
	fmt.Printf("\nReplay summary: total %d, succeeded %d, failed %d\n", len(results), len(results)-errorCount, errorCount)

	return nil
}

// resolveSession finds a session by ID or name
func resolveSession(store *storage.FileStore, nameOrID string) (string, error) {
	// Try using as ID first
	sess, err := store.Get(nameOrID)
	if err == nil {
		return sess.ID, nil
	}

	// Search by name
	sessions, err := store.List(&model.SessionFilter{Name: nameOrID})
	if err != nil {
		return "", fmt.Errorf("failed to query sessions: %w", err)
	}
	if len(sessions) == 0 {
		return "", fmt.Errorf("session not found: %s", nameOrID)
	}
	if len(sessions) > 1 {
		fmt.Printf("Multiple matching sessions found, using the latest: %s (%s)\n", sessions[0].ID, sessions[0].Name)
	}
	return sessions[0].ID, nil
}
