# archive_entries           = true # whether to archive entries in a specified archive directory
# archive_path              = ".archive" # archive path, relative to the directory of the configuration file
changelog_file            = "CHANGELOG.md" # changelog file path, relative to the directory of the configuration file
default_version_increment = "minor" # default semver version increment, attempts to read `changelog_file` to determine the last release
# entry_format              = "hcl" # format of new entry files made by `changeloggy add`: `hcl`, `md`, or `yml`. Entry files in any format are always read

format {
  date = "January 2, 2006"

  template = <<-TEMPLATE
## {{ .Version }} ({{ .Date }})
{{ if .Notes }}
{{ range .Notes }}* {{ .Body }} [GH-{{ .PR }}]
{{ end }}{{ end }}{{ range .Kinds }}
{{ .Heading }}:

{{ range .Entries }}* {{ .Body }} [GH-{{ .PR }}]
{{ end }}{{ end }}
TEMPLATE
}

kind "feature" {
  heading  = "FEATURES"
  priority = 1 # Priority, used in ordering. Duplicate priorities between `kind` blocks are not allowed.

  type "new-resource" {
    regex    = "^\\*\\*New Resource\\*\\*: `\\w+`$" # Optional regex that all entries will be validated against
    example  = "**New Resource**: `azurerm_example`" # If `regex` is specified, `example` must be as well
    priority = 1 # Priority, used in ordering. Duplicate priorities between `type` blocks are not allowed.
  }

  type "new-data-source" {
    regex    = "^\\*\\*New Data Source\\*\\*: `\\w+`$"
    example  = "**New Data Source**: `azurerm_example`"
    priority = 2
  }
}

kind "enhancement" {
  heading  = "ENHANCEMENTS"
  priority = 2

  type "dependency-bump" {
    regex    = "^dependencies: `[^`]+` has been updated from `[^`]+` to `[^`]+`$"
    example  = "dependencies: `github.com/hashicorp/go-azure-sdk` has been updated from `v0.20.0` to `v0.21.0`"
    priority = 1
  }

  type "enhancement" {
    regex    = "^`[^`]+` - .+$"
    example  = "`azurerm_storage_account` - improve validation for the `name` property"
    priority = 2
  }
}

kind "bug" {
  heading  = "BUG FIXES"
  priority = 3

  type "bug" {
    regex    = "^`[^`]+` - .+$"
    example  = "`azurerm_linux_virtual_machine` - fix a crash when `boot_diagnostics` is removed"
    priority = 1
  }
}
