package entryfile

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"

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

// ForPR returns the paths of all entry files in dir for the given PR, in any supported format.
func ForPR(dir string, pr int64) ([]string, error) {
	var paths []string
	for _, ext := range slices.Sorted(maps.Keys(formats)) {
		path := filepath.Join(dir, strconv.FormatInt(pr, 10)+ext)
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("checking for changelog entry file (%s): %w", path, err)
		}
		paths = append(paths, path)
	}
	return paths, nil
}
