package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

const (
	FileName           = ".changeloggy.hcl"
	DefaultEntriesPath = ".changelog"
	DefaultArchivePath = ".archive"
	DefaultEntryFormat = "hcl"
)

type TypeEntry struct {
	EntryType EntryType
	Kind      Kind
}

type Config struct {
	ArchiveEntries          *bool   `hcl:"archive_entries,optional"`
	ArchivePath             *string `hcl:"archive_path,optional"`
	EntriesPath             *string `hcl:"entries_path,optional"`
	EntryFormat             *string `hcl:"entry_format,optional"`
	KindsMap                map[string]Kind
	Types                   map[string]TypeEntry
	ChangelogFile           string `hcl:"changelog_file"`
	DefaultVersionIncrement string `hcl:"default_version_increment"`
	ConfigDir               string
	Format                  []Format `hcl:"format,block"`
	Kinds                   []Kind   `hcl:"kind,block"`
}

func Load(configPath string) (*Config, error) {
	cfg, err := ParseConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("configuration file not found (%s)\nrun `changeloggy config init` to create one", configPath)
		}
		return nil, fmt.Errorf("parsing configuration (%s): %w", configPath, err)
	}
	return cfg, nil
}

func LoadAndValidate(configPath string) (*Config, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}

	if errs := cfg.Validate(); len(errs) > 0 {
		msg := fmt.Sprintf("configuration (%s) is invalid:", configPath)
		var msgSb strings.Builder
		for _, e := range errs {
			msgSb.WriteString("\n  " + e.Error())
		}
		msg += msgSb.String()
		return nil, fmt.Errorf("%s", msg)
	}

	return cfg, nil
}

func ParseConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}

	c := &Config{}
	if err := hclsimple.DecodeFile(configPath, nil, c); err != nil {
		return nil, err
	}

	c.ConfigDir = filepath.Dir(configPath)

	if len(c.Format) == 0 {
		c.Format = append(c.Format, NewDefaultFormat())
	}

	if c.Format[0].DateFormat == nil {
		c.Format[0].DateFormat = new(defaultDateFormat)
	}

	if c.Format[0].Template == nil {
		c.Format[0].Template = new(defaultTemplate)
	}

	km := make(map[string]Kind)
	ti := make(map[string]TypeEntry)
	for i := range c.Kinds {
		c.Kinds[i].BuildTypesMap()
		km[c.Kinds[i].Name] = c.Kinds[i]
		for _, t := range c.Kinds[i].Types {
			ti[t.Name] = TypeEntry{
				Kind:      c.Kinds[i],
				EntryType: t,
			}
		}
	}
	c.KindsMap = km
	c.Types = ti

	return c, nil
}

func (c *Config) Validate() (e []error) {
	l := len(c.Format)
	if l > 1 {
		e = append(e, fmt.Errorf("expected only one `format` block to be specified, got %d", l))
	}

	if l == 1 {
		if errs := c.Format[0].Validate(); len(errs) > 0 {
			e = append(e, errs...)
		}
	}

	switch c.DefaultVersionIncrement {
	case "major", "minor", "patch":
	default:
		e = append(e, fmt.Errorf("invalid `default_version_increment` (`%s`): must be `major`, `minor`, or `patch`", c.DefaultVersionIncrement))
	}

	switch c.EntryFormatOrDefault() {
	case "hcl", "md", "yml":
	default:
		e = append(e, fmt.Errorf("invalid `entry_format` (`%s`): must be `hcl`, `md`, or `yml`", c.EntryFormatOrDefault()))
	}

	e = append(e, c.ValidateKinds()...)

	return
}

func (c *Config) ValidateKinds() (e []error) {
	names := make(map[string]struct{})
	typeNames := make(map[string][]string)
	priorities := make(map[int]string)
	headings := make(map[string]string)

	for _, k := range c.Kinds {
		if _, ok := names[k.Name]; ok {
			e = append(e, fmt.Errorf("duplicate kind (`%s`) defined in config", k.Name))
		}
		names[k.Name] = struct{}{}

		if k.Heading != nil {
			if existing, ok := headings[*k.Heading]; ok {
				e = append(e, fmt.Errorf("duplicate heading (`%s`) defined in both kind `%s` and kind `%s`", *k.Heading, existing, k.Name))
			}
			headings[*k.Heading] = k.Name
		}

		if k.Priority != nil {
			if existing, ok := priorities[*k.Priority]; ok {
				e = append(e, fmt.Errorf("duplicate priority (`%d`) defined in both kind `%s` and kind `%s`", *k.Priority, existing, k.Name))
			}
			priorities[*k.Priority] = k.Name
		}

		e = append(e, k.ValidateTypes(typeNames)...)
	}

	for k, v := range typeNames {
		if len(v) > 1 {
			e = append(e, fmt.Errorf("duplicate type (`%s`) defined in multiple kinds (%s)", k, strings.Join(v, ",")))
		}
	}

	return e
}

func (c *Config) EntriesPathOrDefault() string {
	if c.EntriesPath != nil {
		return c.ResolveRelativePath(*c.EntriesPath)
	}
	return c.ResolveRelativePath(DefaultEntriesPath)
}

func (c *Config) EntryFormatOrDefault() string {
	if c.EntryFormat != nil {
		return *c.EntryFormat
	}
	return DefaultEntryFormat
}

func (c *Config) ArchivePathOrDefault() string {
	if c.ArchivePath != nil {
		return c.ResolveRelativePath(*c.ArchivePath)
	}
	return c.ResolveRelativePath(DefaultArchivePath)
}

func (c *Config) ResolveRelativePath(p string) string {
	if filepath.IsAbs(p) || c.ConfigDir == "" {
		return p
	}
	return filepath.Join(c.ConfigDir, p)
}

func (c *Config) ChangelogFilePath() string {
	return c.ResolveRelativePath(c.ChangelogFile)
}

func (c *Config) ResolveEntryType(name string) (TypeEntry, error) {
	if t, ok := c.Types[name]; ok {
		return t, nil
	}
	return TypeEntry{}, fmt.Errorf("unknown type `%s`, run `changeloggy types` to see available types", name)
}

func (c *Config) EntryTypePriority(kindName, typeName string) int {
	k, ok := c.KindsMap[kindName]
	if !ok {
		return math.MaxInt32
	}
	return k.TypePriority(typeName)
}

func (c *Config) HeadingPriority(heading string) int {
	for _, k := range c.Kinds {
		if k.Heading != nil && *k.Heading == heading {
			if k.Priority != nil {
				return *k.Priority
			}
			return math.MaxInt32
		}
	}
	return math.MaxInt32
}
