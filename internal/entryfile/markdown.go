package entryfile

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sreallymatt/changeloggy/internal/changes"
)

// parseMarkdown reads entries from fenced code blocks. The text after the opening fence is the entry type, and each
// non-blank line inside the block is one entry of that type:
//
//	```bug
//	fix a crash when the config file is empty
//	fix the `--force` flag being ignored
//	```
func parseMarkdown(_ string, src []byte) (*changes.Entries, error) {
	entries := &changes.Entries{}
	typ := "" // type of the block being read, empty when outside a block

	for i, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		info, fence := strings.CutPrefix(line, "```")

		switch {
		case fence:
			info = strings.TrimSpace(strings.TrimLeft(info, "`"))
			switch {
			case typ == "" && info == "":
				return nil, fmt.Errorf("line %d: code block is missing an entry type, e.g. ```bug", i+1)
			case typ == "":
				typ = info
			case info == "":
				typ = ""
			default:
				return nil, fmt.Errorf("line %d: code block for `%s` must be closed before starting a new one", i+1, typ)
			}
		case line == "":
		case typ == "":
			return nil, fmt.Errorf("line %d: entries must be inside a code block, e.g. ```bug", i+1)
		default:
			entries.Add(changes.Entry{Type: typ, Body: line})
		}
	}

	if typ != "" {
		return nil, fmt.Errorf("code block for `%s` is never closed", typ)
	}
	return entries, nil
}

// writeMarkdown writes one code block per entry type, in the order each type first appears.
func writeMarkdown(e *changes.Entries) ([]byte, error) {
	var types []string
	bodies := make(map[string][]string)
	for _, ch := range e.Changes {
		if strings.Contains(ch.Body, "\n") {
			return nil, errors.New("markdown entries must be a single line")
		}
		if _, ok := bodies[ch.Type]; !ok {
			types = append(types, ch.Type)
		}
		bodies[ch.Type] = append(bodies[ch.Type], ch.Body)
	}

	var b strings.Builder
	for i, t := range types {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "```%s\n%s\n```\n", t, strings.Join(bodies[t], "\n"))
	}
	return []byte(b.String()), nil
}
