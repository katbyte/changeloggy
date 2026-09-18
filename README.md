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
| `entry_format`              | Format `changeloggy add` writes entry files in: `hcl` (default), `md`, or `yml`. All formats are read. |

## Entry files

Each pull request gets its own entry file in `entries_path`, named after the PR number (e.g. `.changelog/42.hcl`).
Three formats are supported, and the file extension decides how a file is read, so formats can be mixed.
`entry_format` only picks the format `changeloggy add` writes.

The examples below all hold the same three entries.

### HCL (`42.hcl`)

```hcl
change "cli" {
  body = "cli: `add` - support Markdown and YAML entry files"
}

change "cli" {
  body = "cli: `check` - validate entry files in every format"
}

change "generic-bug" {
  body = "fix a panic caused by dark magic"
}
```

### Markdown (`42.md`)

The text after the opening fence is the entry type, and each line inside the block is one entry of that type.
Blank lines are ignored. Any other text outside a code block is an error.

````markdown
```cli
cli: `add` - support Markdown and YAML entry files
cli: `check` - validate entry files in every format
```

```generic-bug
fix a panic caused by dark magic
```
````

### YAML (`42.yml` or `42.yaml`)

A map of entry type to a list of entries. Quote any entry that starts with a backtick or `*`, or that contains `: `,
otherwise YAML reads those characters as syntax rather than text.

```yaml
cli:
  - "cli: `add` - support Markdown and YAML entry files"
  - "cli: `check` - validate entry files in every format"
generic-bug:
  - fix a panic caused by dark magic
```
