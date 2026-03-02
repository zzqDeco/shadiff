package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"shadiff/internal/model"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage recording sessions",
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	RunE:  runSessionList,
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show session details",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionShow,
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionDelete,
}

var (
	sessionTagFilter string
)

func init() {
	sessionListCmd.Flags().StringVar(&sessionTagFilter, "tag", "", "Filter by tag")

	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)
	rootCmd.AddCommand(sessionCmd)
}

func getStore() (*storage.FileStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	dataDir := homeDir + "/.shadiff"
	return storage.NewFileStore(dataDir)
}

func runSessionList(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	var filter *model.SessionFilter
	if sessionTagFilter != "" {
		filter = &model.SessionFilter{Tags: []string{sessionTagFilter}}
	}

	sessions, err := store.List(filter)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tRECORDS\tCREATED")
	for _, s := range sessions {
		created := time.UnixMilli(s.CreatedAt).Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", s.ID, s.Name, s.Status, s.RecordCount, created)
	}
	return w.Flush()
}

func runSessionShow(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	sess, err := store.Get(args[0])
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	fmt.Printf("ID:          %s\n", sess.ID)
	fmt.Printf("Name:        %s\n", sess.Name)
	fmt.Printf("Description: %s\n", sess.Description)
	fmt.Printf("Status:      %s\n", sess.Status)
	fmt.Printf("Records:     %d\n", sess.RecordCount)
	fmt.Printf("Source:      %s\n", sess.Source.BaseURL)
	fmt.Printf("Target:      %s\n", sess.Target.BaseURL)
	fmt.Printf("Tags:        %v\n", sess.Tags)
	fmt.Printf("Created:     %s\n", time.UnixMilli(sess.CreatedAt).Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", time.UnixMilli(sess.UpdatedAt).Format("2006-01-02 15:04:05"))

	return nil
}

func runSessionDelete(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	// Verify session exists first
	sess, err := store.Get(args[0])
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if err := store.Delete(sess.ID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	fmt.Printf("Session deleted: %s (%s)\n", sess.ID, sess.Name)
	return nil
}
