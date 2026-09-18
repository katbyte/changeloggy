package changes

import (
	"fmt"
	"regexp"

	"github.com/sreallymatt/changeloggy/internal/config"
)

type Entries struct {
	Changes []Entry `hcl:"change,block"`
}

type Entry struct {
	Type string `hcl:"type,label"`
	Body string `hcl:"body"`

	Kind string
	PR   int64
}

type WriteEntry struct {
	Type string `hcl:"type,label"`
	Body string `hcl:"body"`
}

type WriteEntries struct {
	Changes []WriteEntry `hcl:"change,block"`
}

func (e *Entries) Add(input Entry) {
	e.Changes = append(e.Changes, input)
}

func (e *Entries) WriteEntries() WriteEntries {
	result := WriteEntries{}
	for _, ch := range e.Changes {
		result.Changes = append(result.Changes, WriteEntry{
			Type: ch.Type,
			Body: ch.Body,
		})
	}
	return result
}

func (e *Entry) Validate(t config.EntryType) error {
	if t.Regex == nil {
		return nil
	}

	re, err := regexp.Compile(*t.Regex)
	if err != nil {
		return fmt.Errorf("received an invalid regular expression (type `%s`): %w", e.Type, err)
	}

	if !re.MatchString(e.Body) {
		return fmt.Errorf("body does not match expected format (type %q)\n\nreceived: %q\nexample:  %q\n\nregex: %s", e.Type, e.Body, *t.Example, re.String())
	}

	return nil
}
