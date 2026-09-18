package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/spf13/cobra"
	"github.com/sreallymatt/changeloggy/internal/changes"
	"github.com/sreallymatt/changeloggy/internal/config"
	"github.com/sreallymatt/changeloggy/internal/entryfile"
	"github.com/sreallymatt/changeloggy/internal/templatehelper"
	"github.com/sreallymatt/changeloggy/internal/util"
)

const root string = "__ROOT"

func NewGenerateCommand(configPath *string) *cobra.Command {
	var version string

	c := &cobra.Command{
		Use:   "generate",
		Short: "Generates a new changelog.",
		Long:  `Generates a new changelog entry for a release, prepending it to the changelog file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadAndValidate(*configPath)
			if err != nil {
				return err
			}

			if version == "" {
				version, err = cfg.NextVersion()
				if err != nil {
					return fmt.Errorf("determining next version: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "No --version provided, using auto-bumped version: %s\n", version)
			}

			grouped, files, err := parseChangesFromDirectory(cmd, cfg, cfg.EntriesPathOrDefault())
			if err != nil {
				return err
			}

			if len(grouped) == 0 {
				return errors.New("no changelog entries found, nothing to generate")
			}

			changelogPath := cfg.ChangelogFilePath()

			var existing []byte
			if _, err := os.Stat(changelogPath); !errors.Is(err, os.ErrNotExist) {
				existing, err = os.ReadFile(filepath.Clean(changelogPath))
				if err != nil {
					return fmt.Errorf("reading file (%s): %w", changelogPath, err)
				}
			}

			formatted, err := formatWithTemplate(cfg, grouped, version)
			if err != nil {
				return fmt.Errorf("rendering changelog: %w", err)
			}

			if err := atomicWrite(changelogPath, []byte(formatted+string(existing))); err != nil {
				return fmt.Errorf("writing changelog (%s): %w", changelogPath, err)
			}

			if pointer.From(cfg.ArchiveEntries) {
				if err := archiveEntries(cfg, files); err != nil {
					return fmt.Errorf("archiving entries: %w", err)
				}
			} else {
				if err := removeEntries(files); err != nil {
					return fmt.Errorf("removing entries: %w", err)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generated %s -> %s\n", version, changelogPath)
			return nil
		},
	}

	c.Flags().StringVarP(&version, "version", "v", "", "The version of this release.")

	return c
}

func atomicWrite(dst string, content []byte) error {
	if err := util.EnsureDir(dst); err != nil {
		return err
	}

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".changeloggy-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmp.Name(), dst); err != nil {
		return fmt.Errorf("renaming temp file to destination: %w", err)
	}
	return nil
}

// parseChangesFromDirectory returns the entries grouped by heading and priority, along with the paths of the files
// they were read from.
func parseChangesFromDirectory(cmd *cobra.Command, cfg *config.Config, directory string) (grouped map[string]map[int]*changes.Entries, parsed []string, err error) {
	grouped = make(map[string]map[int]*changes.Entries)

	files, skipped, err := entryfile.List(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return grouped, nil, nil
		}
		return nil, nil, err
	}

	for _, name := range skipped {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping %s: filename is not a valid PR number\n", name)
	}

	for _, f := range files {
		if err := parseChangesFromFile(cfg, f, grouped); err != nil {
			return nil, nil, err
		}
		parsed = append(parsed, f.Path)
	}

	return grouped, parsed, nil
}

func parseChangesFromFile(cfg *config.Config, file entryfile.File, result map[string]map[int]*changes.Entries) error {
	entries, err := entryfile.Read(file.Path)
	if err != nil {
		return err
	}

	for i, change := range entries.Changes {
		t, err := cfg.ResolveEntryType(change.Type)
		if err != nil {
			return fmt.Errorf("file (%s) entry %d: %w", file.Path, i+1, err)
		}

		if err := change.Validate(t.EntryType); err != nil {
			return fmt.Errorf("file (%s) entry %d (%s): %w", file.Path, i+1, change.Type, err)
		}

		heading := root
		if t.Kind.Heading != nil {
			heading = *t.Kind.Heading
		}

		if _, ok := result[heading]; !ok {
			result[heading] = make(map[int]*changes.Entries)
		}

		priority := cfg.EntryTypePriority(t.Kind.Name, change.Type)
		if result[heading][priority] == nil {
			result[heading][priority] = &changes.Entries{}
		}

		entries.Changes[i].PR = file.PR
		entries.Changes[i].Kind = t.Kind.Name
		result[heading][priority].Add(entries.Changes[i])
	}

	return nil
}

func buildReleaseData(cfg *config.Config, grouped map[string]map[int]*changes.Entries, version, timestamp string) templatehelper.ReleaseData {
	orderedKinds := make([]string, 0, len(grouped))
	for k := range grouped {
		if k != root {
			orderedKinds = append(orderedKinds, k)
		}
	}
	slices.SortFunc(orderedKinds, func(a, b string) int {
		if cfg.HeadingPriority(a) != cfg.HeadingPriority(b) {
			return cfg.HeadingPriority(a) - cfg.HeadingPriority(b)
		}
		return strings.Compare(a, b)
	})

	kinds := make([]templatehelper.KindData, 0, len(orderedKinds))
	for _, kind := range orderedKinds {
		kinds = append(kinds, buildKindData(kind, grouped[kind]))
	}

	var notes []templatehelper.EntryData
	if rootBucket, ok := grouped[root]; ok {
		notes = buildKindData(root, rootBucket).Entries
	}

	return templatehelper.ReleaseData{
		Version: version,
		Date:    timestamp,
		Kinds:   kinds,
		Notes:   notes,
	}
}

func buildKindData(kind string, bucket map[int]*changes.Entries) templatehelper.KindData {
	orderedPriority := make([]int, 0, len(bucket))
	for k := range bucket {
		orderedPriority = append(orderedPriority, k)
	}
	slices.Sort(orderedPriority)

	entries := make([]templatehelper.EntryData, 0)
	for _, priority := range orderedPriority {
		rows := bucket[priority].Changes
		slices.SortFunc(rows, func(a, b changes.Entry) int {
			return strings.Compare(a.Body, b.Body)
		})
		for _, e := range rows {
			entries = append(entries, templatehelper.EntryData{
				PR:   e.PR,
				Body: e.Body,
				Type: e.Type,
				Kind: e.Kind,
			})
		}
	}

	return templatehelper.KindData{
		Heading: kind,
		Entries: entries,
	}
}

func formatWithTemplate(cfg *config.Config, grouped map[string]map[int]*changes.Entries, version string) (string, error) {
	return templatehelper.Render(*cfg.Format[0].Template, buildReleaseData(cfg, grouped, version, time.Now().Format(*cfg.Format[0].DateFormat)))
}

func archiveEntries(cfg *config.Config, files []string) error {
	archivePath := cfg.ArchivePathOrDefault()

	if err := os.MkdirAll(archivePath, 0o700); err != nil {
		return fmt.Errorf("creating archive directory (%s): %w", archivePath, err)
	}

	for _, src := range files {
		dst := filepath.Join(archivePath, filepath.Base(src))
		if err := moveFile(src, dst); err != nil {
			return fmt.Errorf("archiving file (%s): %w", filepath.Base(src), err)
		}
	}

	return nil
}

func removeEntries(files []string) error {
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return fmt.Errorf("removing file (%s): %w", filepath.Base(f), err)
		}
	}

	return nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(filepath.Clean(src))
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return fmt.Errorf("copying file: %w", err)
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("closing destination file: %w", err)
	}

	return os.Remove(src)
}
