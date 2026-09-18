package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"
	"github.com/sreallymatt/changeloggy/internal/changes"
	"github.com/sreallymatt/changeloggy/internal/config"
	"github.com/sreallymatt/changeloggy/internal/hclparse"
	"github.com/sreallymatt/changeloggy/internal/util"
)

func NewAddCommand(configPath *string) *cobra.Command {
	var entryType string
	var pr int64
	var replace bool

	c := &cobra.Command{
		Use:   "add --pr <num> --type <type> <body>",
		Short: "Adds a new changelog entry.",
		Long: `Usage: changeloggy add --pr integer --type string [flags] <body>

Adds a new changelog entry for a pull request.

If no changelog file exists for the pull request number, one will be created.

Run 'changeloggy types' to see all available entry types.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadAndValidate(*configPath)
			if err != nil {
				return fmt.Errorf("loading config (%s): %w", *configPath, err)
			}

			body := args[0]

			t, err := cfg.ResolveEntryType(entryType)
			if err != nil {
				return err
			}

			ch := changes.Entry{
				Type: entryType,
				Kind: t.Kind.Name,
				Body: body,
			}

			if err := ch.Validate(t.EntryType); err != nil {
				return fmt.Errorf("invalid changelog entry: %w", err)
			}

			filePath := filepath.Join(cfg.EntriesPathOrDefault(), fmt.Sprintf("%d.hcl", pr))

			if err := util.EnsureDir(filePath); err != nil {
				return err
			}

			_, err = os.Stat(filePath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("checking for presence of file (%s): %w", filePath, err)
			}

			chs := changes.Entries{}
			if !replace && !errors.Is(err, os.ErrNotExist) {
				existing, err := hclparse.EntryFile(filePath)
				if err != nil {
					return err
				}
				chs = *existing
			}

			chs.Changes = append(chs.Changes, ch)

			newContent := hclwrite.NewFile()
			gohcl.EncodeIntoBody(chs.WriteEntries(), newContent.Body())

			if err := os.WriteFile(filePath, bytes.TrimLeft(hclwrite.Format(newContent.Bytes()), "\n"), 0o600); err != nil {
				return fmt.Errorf("writing to file (%s): %w", filePath, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "added `%s` entry to `%s`\n", entryType, filePath)
			return nil
		},
	}

	c.Flags().Int64VarP(&pr, "pr", "p", -1, "The PR number for this change.")
	c.Flags().BoolVarP(&replace, "replace", "r", false, "Replace existing changelog entry file.")
	c.Flags().StringVarP(&entryType, "type", "t", "", "The entry type.")

	_ = c.MarkFlagRequired("pr")
	_ = c.MarkFlagRequired("type")

	return c
}
