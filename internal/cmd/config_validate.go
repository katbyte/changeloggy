package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sreallymatt/changeloggy/internal/config"
)

func NewConfigValidateCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validates the configuration file.",
		Long: `Validates the changeloggy configuration file.

Checks for:
  - Duplicate kind names in config
  - Duplicate type names across kinds
  - Invalid regex patterns
  - Missing examples where regex is set
  - Priority conflicts within the config (two kinds sharing the same priority)
  - Priority conflicts within a kind (two types sharing the same priority)
  - Template syntax errors
  - Unsupported entry_format`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*configPath)
			if err != nil {
				return err
			}

			if errs := cfg.Validate(); len(errs) > 0 {
				msg := fmt.Sprintf("configuration (%s) is invalid:", *configPath)
				var msgSb strings.Builder
				for _, e := range errs {
					msgSb.WriteString("\n  " + e.Error())
				}
				msg += msgSb.String()
				return fmt.Errorf("%s", msg)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "configuration (%s) is valid\n", *configPath)
			return nil
		},
	}
}
