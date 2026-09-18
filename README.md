# changeloggy

A CLI tool for managing changelog entries with optional entry format validation.

## Installation

```sh
go install github.com/sreallymatt/changeloggy@latest
```

## Usage

Initialize a config file in your repository:

```sh
changeloggy config init
```

Add a changelog entry for a pull request:

```sh
changeloggy add --pr 42 --type example "Feat: an example entry"
```

Validate entries:

```sh
changeloggy check --pr 42 # validate a single PR entry
changeloggy check          # validate all entries
```

Generate the changelog:

```sh
changeloggy generate --version 1.2.0
changeloggy generate # auto-bumps version based on `default_version_increment`
```

## Configuration

Run `changeloggy config init` to create a `.changeloggy.hcl` config file. Key options:

| Option                      | Description                                                                                            |
|-----------------------------|--------------------------------------------------------------------------------------------------------|
| `changelog_file`            | Path to the changelog file                                                                             |
| `default_version_increment` | `major`, `minor`, or `patch`                                                                           |
| `archive_entries`           | If `true`, moves entries to `archive_path` after generation. If `false` or unset, entries are deleted. |
| `entries_path`              | Directory where entry files are stored (default: `.changelog/`)                                        |
| `entry_format`              | Format of new entry files made by `changeloggy add`: `hcl` (default), `md`, or `yml`.                  |

## Entry files

Each pull request gets one entry file in `entries_path`, named after the PR number (e.g. `.changelog/77.hcl`).
Three formats are supported, and the file extension decides how a file is read, so different PRs can use different
formats. A PR with more than one entry file (e.g. both `77.hcl` and `77.md`) is an error.

`changeloggy add` adds to a PR's existing entry file in whatever format it is in, and only uses `entry_format` when it
has to create a new one.

The examples below all hold the same three entries, using types from the starter config that `changeloggy config init`
creates.

### HCL (`77.hcl`)

```hcl
change "enhancement" {
  body = "`azurerm_kubernetes_cluster` - support for the `node_provisioning_profile` block"
}

change "enhancement" {
  body = "`azurerm_storage_account` - improve validation for the `name` property"
}

change "bug" {
  body = "`azurerm_linux_virtual_machine` - fix a crash when `boot_diagnostics` is removed"
}
```

### Markdown (`77.md`)

The text after the opening fence is the entry type, and each line inside the block is one entry of that type.
Blank lines are ignored. Any other text outside a code block is an error.

````markdown
```enhancement
`azurerm_kubernetes_cluster` - support for the `node_provisioning_profile` block
`azurerm_storage_account` - improve validation for the `name` property
```

```bug
`azurerm_linux_virtual_machine` - fix a crash when `boot_diagnostics` is removed
```
````

### YAML (`77.yml` or `77.yaml`)

A map of entry type to a list of entries. Quote any entry that starts with a backtick or `*`, or that contains `: `,
otherwise YAML reads those characters as syntax rather than text.

```yaml
enhancement:
  - "`azurerm_kubernetes_cluster` - support for the `node_provisioning_profile` block"
  - "`azurerm_storage_account` - improve validation for the `name` property"
bug:
  - "`azurerm_linux_virtual_machine` - fix a crash when `boot_diagnostics` is removed"
```
