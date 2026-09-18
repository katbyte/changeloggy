package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sreallymatt/changeloggy/internal/config"
)

func NewTypesCommand(configPath *string) *cobra.Command {
	var table bool

	c := &cobra.Command{
		Use:   "types",
		Short: "Lists all available entry types.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadAndValidate(*configPath)
			if err != nil {
				return err
			}

			if table {
				return printTypesTable(cmd, cfg)
			}
			return printTypesList(cmd, cfg)
		},
	}

	c.Flags().BoolVarP(&table, "table", "t", false, "Output types as a Markdown table.")

	return c
}

func printTypesList(cmd *cobra.Command, cfg *config.Config) error {
	for _, k := range cfg.Kinds {
		heading := "(no heading)"
		if k.Heading != nil {
			heading = *k.Heading
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s\n", k.Name, heading)

		if len(k.Types) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  (no types defined)\n\n")
			continue
		}

		for _, t := range k.Types {
			line := strings.Builder{}
			fmt.Fprintf(&line, "  %-24s", t.Name)
			if t.Example != nil {
				fmt.Fprintf(&line, "  e.g. %s", *t.Example)
			}
			fmt.Fprintln(cmd.OutOrStdout(), line.String())
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

func printTypesTable(cmd *cobra.Command, cfg *config.Config) error {
	fmt.Fprintf(cmd.OutOrStdout(), "| Section | Type | Example |\n")
	fmt.Fprintf(cmd.OutOrStdout(), "|---------|------|---------|\n")

	for _, k := range cfg.Kinds {
		heading := "(no heading)"
		if k.Heading != nil {
			heading = *k.Heading
		}

		if len(k.Types) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "| %s | (no types defined) | |\n", heading)
			continue
		}

		for _, t := range k.Types {
			example := ""
			if t.Example != nil {
				example = *t.Example
			}
			fmt.Fprintf(cmd.OutOrStdout(), "| %s | %s | %s |\n", heading, t.Name, example)
		}
	}
	return nil
}
