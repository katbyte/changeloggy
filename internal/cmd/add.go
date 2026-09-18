package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sreallymatt/changeloggy/internal/changes"
	"github.com/sreallymatt/changeloggy/internal/config"
	"github.com/sreallymatt/changeloggy/internal/entryfile"
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

If the pull request already has a changelog file, the entry is added to it in
whatever format it is in. Otherwise one is created using the configured
'entry_format' (hcl, md, or yml).

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

			entriesDir := cfg.EntriesPathOrDefault()

			filePath, err := entryfile.ForPR(entriesDir, pr)
			if err != nil {
				return err
			}

			chs := changes.Entries{}
			switch {
			case filePath == "":
				filePath = filepath.Join(entriesDir, fmt.Sprintf("%d.%s", pr, cfg.EntryFormatOrDefault()))
			case !replace:
				existing, err := entryfile.Read(filePath)
				if err != nil {
					return err
				}
				chs = *existing
			}

			if err := util.EnsureDir(filePath); err != nil {
				return err
			}

			chs.Changes = append(chs.Changes, ch)

			if err := entryfile.Write(filePath, &chs); err != nil {
				return err
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
