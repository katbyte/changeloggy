package entryfile

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/sreallymatt/changeloggy/internal/changes"
)

type format struct {
	parse func(filename string, src []byte) (*changes.Entries, error)
	write func(e *changes.Entries) ([]byte, error)
}

// formats maps each supported entry file extension to how it is read and written.
var formats = map[string]format{
	".hcl":  {parse: parseHCL, write: writeHCL},
	".md":   {parse: parseMarkdown, write: writeMarkdown},
	".yml":  {parse: parseYAML, write: writeYAML},
	".yaml": {parse: parseYAML, write: writeYAML},
}

// Supported reports whether name has the extension of a supported entry file format.
func Supported(name string) bool {
	_, ok := formats[filepath.Ext(name)]
	return ok
}

// Read parses the entry file at path, using its extension to pick the format.
func Read(path string) (*changes.Entries, error) {
	f, ok := formats[filepath.Ext(path)]
	if !ok {
		return nil, fmt.Errorf("unsupported changelog entry file extension (%s)", path)
	}

	src, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading changelog entry file (%s): %w", path, err)
	}

	entries, err := f.parse(path, src)
	if err != nil {
		return nil, fmt.Errorf("parsing changelog entry file (%s): %w", path, err)
	}
	return entries, nil
}

// Write encodes entries in the format matching the extension of path, and writes them to path.
func Write(path string, e *changes.Entries) error {
	f, ok := formats[filepath.Ext(path)]
	if !ok {
		return fmt.Errorf("unsupported changelog entry file extension (%s)", path)
	}

	src, err := f.write(e)
	if err != nil {
		return fmt.Errorf("encoding changelog entry file (%s): %w", path, err)
	}

	if err := os.WriteFile(path, src, 0o600); err != nil {
		return fmt.Errorf("writing to file (%s): %w", path, err)
	}
	return nil
}

// File is an entry file and the PR it belongs to.
type File struct {
	Path string
	PR   int64
}

// ForPR returns the path of the entry file in dir for the given PR, or an empty string if there isn't one. It is an
// error for a PR to have more than one entry file.
func ForPR(dir string, pr int64) (string, error) {
	var paths []string
	for _, ext := range slices.Sorted(maps.Keys(formats)) {
		path := filepath.Join(dir, strconv.FormatInt(pr, 10)+ext)
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("checking for changelog entry file (%s): %w", path, err)
		}
		paths = append(paths, path)
	}

	switch len(paths) {
	case 0:
		return "", nil
	case 1:
		return paths[0], nil
	default:
		return "", duplicateError(pr, paths)
	}
}

// List returns the entry files in dir, in name order. Files with a supported extension that are not named after a PR
// number are returned in skipped. It is an error for a PR to have more than one entry file.
func List(dir string) (files []File, skipped []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading entries directory (%s): %w", dir, err)
	}

	paths := make(map[int64][]string)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !Supported(name) {
			continue
		}

		pr, err := strconv.ParseInt(strings.TrimSuffix(name, filepath.Ext(name)), 10, 64)
		if err != nil || pr < 1 {
			skipped = append(skipped, name)
			continue
		}

		path := filepath.Join(dir, name)
		paths[pr] = append(paths[pr], path)
		files = append(files, File{Path: path, PR: pr})
	}

	for _, f := range files {
		if len(paths[f.PR]) > 1 {
			return nil, nil, duplicateError(f.PR, paths[f.PR])
		}
	}

	return files, skipped, nil
}

func duplicateError(pr int64, paths []string) error {
	return fmt.Errorf("PR #%d has more than one changelog entry file (%s), combine them into one", pr, strings.Join(paths, ", "))
}
