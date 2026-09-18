package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sreallymatt/changeloggy/internal/config"
	"github.com/sreallymatt/changeloggy/internal/entryfile"
)

func NewCheckCommand(configPath *string) *cobra.Command {
	var pr int64

	c := &cobra.Command{
		Use:   "check [--pr <num>]",
		Short: "Validates changelog entry file(s).",
		Long: `Validates changelog entry files.

When --pr is provided, validates the entry file for that pull request.
When omitted, validates all entry files in the entries directory.

All entry types must be valid and all bodies must match their configured regex.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadAndValidate(*configPath)
			if err != nil {
				return err
			}

			if pr == -1 {
				return checkAll(cmd, cfg)
			}
			return checkPR(cmd, cfg, pr)
		},
	}

	c.Flags().Int64VarP(&pr, "pr", "p", -1, "The PR number to check. If omitted, all entries are checked.")

	return c
}

func checkPR(cmd *cobra.Command, cfg *config.Config, pr int64) error {
	entriesDir := cfg.EntriesPathOrDefault()

	filePath, err := entryfile.ForPR(entriesDir, pr)
	if err != nil {
		return err
	}

	if filePath == "" {
		expected := filepath.Join(entriesDir, fmt.Sprintf("%d.%s", pr, cfg.EntryFormatOrDefault()))
		return fmt.Errorf("no changelog entry found for PR #%d (expected %s)", pr, expected)
	}

	count, err := validateFile(cfg, filePath)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "changelog entry for PR #%d is valid (%s)\n", pr, entryCount(count))
	return nil
}

func checkAll(cmd *cobra.Command, cfg *config.Config) error {
	entriesDir := cfg.EntriesPathOrDefault()

	files, skipped, err := entryfile.List(entriesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("entries directory not found (%s)", entriesDir)
		}
		return err
	}

	warnSkipped(cmd, skipped)

	if len(files) == 0 {
		return fmt.Errorf("no changelog entry files found in %s", entriesDir)
	}

	totalEntries := 0
	for _, f := range files {
		count, err := validateFile(cfg, f.Path)
		if err != nil {
			return err
		}
		totalEntries += count
	}

	fmt.Fprintf(cmd.OutOrStdout(), "all files valid (%s across %d file(s))\n", entryCount(totalEntries), len(files))
	return nil
}

func warnSkipped(cmd *cobra.Command, skipped []string) {
	for _, name := range skipped {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping %s: filename is not a valid PR number\n", name)
	}
}

func validateFile(cfg *config.Config, filePath string) (int, error) {
	entries, err := entryfile.Read(filePath)
	if err != nil {
		return 0, err
	}

	if len(entries.Changes) == 0 {
		return 0, fmt.Errorf("%s: contains no entries", filePath)
	}

	for i, change := range entries.Changes {
		t, err := cfg.ResolveEntryType(change.Type)
		if err != nil {
			return 0, fmt.Errorf("%s: entry %d - %w", filePath, i+1, err)
		}

		if err := change.Validate(t.EntryType); err != nil {
			return 0, fmt.Errorf("%s: entry %d (`%s`) - %w", filePath, i+1, change.Type, err)
		}
	}

	return len(entries.Changes), nil
}

func entryCount(n int) string {
	suffix := "ies"
	if n == 1 {
		suffix = "y"
	}
	return fmt.Sprintf("%d entr%s", n, suffix)
}
