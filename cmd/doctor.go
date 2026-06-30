package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"shadiff/internal/diagnostics"

	"github.com/spf13/cobra"
)

var (
	doctorFormat string
	doctorStrict bool
	doctorE2E    bool

	doctorCommandRunner diagnostics.CommandRunner
	doctorE2EAddrs      []string
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Shadiff environment readiness",
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().StringVar(&doctorFormat, "format", "terminal", "Output format: terminal, json")
	doctorCmd.Flags().BoolVar(&doctorStrict, "strict", false, "Treat warnings as failures")
	doctorCmd.Flags().BoolVar(&doctorE2E, "e2e", false, "Include official E2E demo port checks")
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	format := strings.ToLower(doctorFormat)
	opts := diagnostics.Options{
		Strict:     doctorStrict,
		IncludeE2E: doctorE2E,
		ConfigPath: cfgFile,
		Version:    Version,
		Commit:     Commit,
		BuildDate:  BuildDate,
		Command:    doctorCommandRunner,
		E2EAddrs:   doctorE2EAddrs,
	}
	report, err := diagnostics.BuildReport(cmd.Context(), opts)
	if err != nil {
		return err
	}

	switch format {
	case "", "terminal":
		diagnostics.PrintReport(cmd.OutOrStdout(), report)
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("write doctor JSON: %w", err)
		}
	default:
		return fmt.Errorf("invalid doctor output format %q: must be terminal or json", doctorFormat)
	}

	if report.Summary.Error > 0 {
		return fmt.Errorf("doctor found %d error check(s)", report.Summary.Error)
	}
	if report.Strict && report.Summary.Warn > 0 {
		return fmt.Errorf("doctor found %d warning check(s) in strict mode", report.Summary.Warn)
	}
	return nil
}
