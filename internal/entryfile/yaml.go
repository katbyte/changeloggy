package entryfile

import (
	"bytes"
	"maps"
	"slices"

	"github.com/sreallymatt/changeloggy/internal/changes"
	"go.yaml.in/yaml/v3"
)

// parseYAML reads entries from a map of entry type to a list of entries of that type. Types are read in alphabetical
// order:
//
//	bug:
//	  - fix a crash when the config file is empty
//	  - "`--force` is no longer ignored"
func parseYAML(_ string, src []byte) (*changes.Entries, error) {
	var bodies map[string][]string
	if err := yaml.Unmarshal(src, &bodies); err != nil {
		return nil, err
	}

	entries := &changes.Entries{}
	for _, t := range slices.Sorted(maps.Keys(bodies)) {
		for _, body := range bodies[t] {
			entries.Add(changes.Entry{Type: t, Body: body})
		}
	}
	return entries, nil
}

func writeYAML(e *changes.Entries) ([]byte, error) {
	bodies := make(map[string][]string)
	for _, ch := range e.Changes {
		bodies[ch.Type] = append(bodies[ch.Type], ch.Body)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(bodies); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
