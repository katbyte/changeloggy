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

	paths, err := entryfile.ForPR(entriesDir, pr)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		expected := filepath.Join(entriesDir, fmt.Sprintf("%d.%s", pr, cfg.EntryFormatOrDefault()))
		return fmt.Errorf("no changelog entry found for PR #%d (expected %s)", pr, expected)
	}

	totalEntries := 0
	for _, path := range paths {
		count, err := validateFile(cfg, path)
		if err != nil {
			return err
		}
		totalEntries += count
	}

	fmt.Fprintf(cmd.OutOrStdout(), "changelog entry for PR #%d is valid (%s)\n", pr, entryCount(totalEntries))
	return nil
}

func checkAll(cmd *cobra.Command, cfg *config.Config) error {
	entriesDir := cfg.EntriesPathOrDefault()

	files, err := os.ReadDir(entriesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("entries directory not found (%s)", entriesDir)
		}
		return fmt.Errorf("reading entries directory (%s): %w", entriesDir, err)
	}

	var entryFiles []string
	for _, f := range files {
		if f.IsDir() || !entryfile.Supported(f.Name()) {
			continue
		}
		if pr, err := prFromFilename(f.Name()); err != nil || pr < 1 {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping %s: filename is not a valid PR number\n", f.Name())
			continue
		}
		entryFiles = append(entryFiles, filepath.Join(entriesDir, f.Name()))
	}

	if len(entryFiles) == 0 {
		return fmt.Errorf("no changelog entry files found in %s", entriesDir)
	}

	totalEntries := 0
	for _, path := range entryFiles {
		count, err := validateFile(cfg, path)
		if err != nil {
			return err
		}
		totalEntries += count
	}

	fmt.Fprintf(cmd.OutOrStdout(), "all files valid (%s across %d file(s))\n", entryCount(totalEntries), len(entryFiles))
	return nil
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
