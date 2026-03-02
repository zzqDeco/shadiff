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
	Short: "管理录制会话",
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有会话",
	RunE:  runSessionList,
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "显示会话详情",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionShow,
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "删除会话",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionDelete,
}

var (
	sessionTagFilter string
)

func init() {
	sessionListCmd.Flags().StringVar(&sessionTagFilter, "tag", "", "按标签过滤")

	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)
	rootCmd.AddCommand(sessionCmd)
}

func getStore() (*storage.FileStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
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
		return fmt.Errorf("列出会话失败: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("暂无会话记录")
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
		return fmt.Errorf("获取会话失败: %w", err)
	}

	fmt.Printf("ID:          %s\n", sess.ID)
	fmt.Printf("名称:        %s\n", sess.Name)
	fmt.Printf("描述:        %s\n", sess.Description)
	fmt.Printf("状态:        %s\n", sess.Status)
	fmt.Printf("记录数:      %d\n", sess.RecordCount)
	fmt.Printf("源端:        %s\n", sess.Source.BaseURL)
	fmt.Printf("目标端:      %s\n", sess.Target.BaseURL)
	fmt.Printf("标签:        %v\n", sess.Tags)
	fmt.Printf("创建时间:    %s\n", time.UnixMilli(sess.CreatedAt).Format("2006-01-02 15:04:05"))
	fmt.Printf("更新时间:    %s\n", time.UnixMilli(sess.UpdatedAt).Format("2006-01-02 15:04:05"))

	return nil
}

func runSessionDelete(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	// 先确认会话存在
	sess, err := store.Get(args[0])
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}

	if err := store.Delete(sess.ID); err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}

	fmt.Printf("已删除会话: %s (%s)\n", sess.ID, sess.Name)
	return nil
}
