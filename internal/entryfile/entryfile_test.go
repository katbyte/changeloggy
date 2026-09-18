package entryfile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sreallymatt/changeloggy/internal/changes"
)

func TestRead(t *testing.T) {
	t.Parallel()

	want := []changes.Entry{
		{Type: "bug", Body: "fix a crash when the config file is empty"},
		{Type: "bug", Body: "`--force` is no longer ignored"},
		{Type: "deps", Body: "`github.com/hashicorp/hcl/v2` has been updated from `v2.23.0` to `v2.24.0`"},
	}

	cases := map[string]string{
		"1.hcl": `
change "bug" {
  body = "fix a crash when the config file is empty"
}

change "bug" {
  body = "` + "`--force`" + ` is no longer ignored"
}

change "deps" {
  body = "` + "`github.com/hashicorp/hcl/v2` has been updated from `v2.23.0` to `v2.24.0`" + `"
}
`,
		"1.md": "```bug\r\n" +
			"fix a crash when the config file is empty\r\n" +
			"\r\n" +
			"  `--force` is no longer ignored  \r\n" +
			"```\r\n" +
			"\r\n" +
			"````deps\r\n" +
			"`github.com/hashicorp/hcl/v2` has been updated from `v2.23.0` to `v2.24.0`\r\n" +
			"````\r\n",
		"1.yml": `
deps:
  - "` + "`github.com/hashicorp/hcl/v2` has been updated from `v2.23.0` to `v2.24.0`" + `"
bug:
  - fix a crash when the config file is empty
  - "` + "`--force`" + ` is no longer ignored"
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := Read(writeTemp(t, name, src))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got.Changes, want) {
				t.Fatalf("\ngot:  %+v\nwant: %+v", got.Changes, want)
			}
		})
	}
}

func TestReadErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, src, wantErr string
	}{
		{"outside-block.md", "fix a thing\n", "line 1: entries must be inside a code block"},
		{"missing-type.md", "```\nfix a thing\n```\n", "line 1: code block is missing an entry type"},
		{"unclosed.md", "```cli\nfix a thing\n", "code block for `cli` is never closed"},
		{"opened-twice.md", "```cli\nfix a thing\n```deps\n", "line 3: code block for `cli` must be closed"},
		{"not-a-list.yml", "cli: fix a thing\n", "cannot unmarshal !!str"},
		{"not-a-map.yml", "- cli\n", "cannot unmarshal !!seq"},
		{"unsupported.txt", "", "unsupported changelog entry file extension"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := Read(writeTemp(t, tc.name, tc.src))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestWriteThenRead(t *testing.T) {
	t.Parallel()

	want := []changes.Entry{
		{Type: "enhancement", Body: "`azurerm_example` - support for the `sku` property"},
		{Type: "enhancement", Body: "`azurerm_example` - improve validation for the `name` property"},
		{Type: "new-resource", Body: "**New Resource**: `azurerm_example`"},
	}

	for _, ext := range []string{".hcl", ".md", ".yml", ".yaml"} {
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "1"+ext)
			if err := Write(path, &changes.Entries{Changes: want}); err != nil {
				t.Fatalf("unexpected error writing: %v", err)
			}

			got, err := Read(path)
			if err != nil {
				t.Fatalf("unexpected error reading: %v", err)
			}
			if !slices.Equal(got.Changes, want) {
				t.Fatalf("\ngot:  %+v\nwant: %+v", got.Changes, want)
			}
		})
	}
}

func TestWriteMarkdownMultiLine(t *testing.T) {
	t.Parallel()

	err := Write(filepath.Join(t.TempDir(), "1.md"), &changes.Entries{Changes: []changes.Entry{{Type: "cli", Body: "one\ntwo"}}})
	if err == nil || !strings.Contains(err.Error(), "must be a single line") {
		t.Fatalf("expected single line error, got: %v", err)
	}
}

func TestForPR(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"1.hcl", "1.md", "1.txt", "12.md", "2.yml"} {
		writeFile(t, filepath.Join(dir, name), "")
	}

	got, err := ForPR(dir, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{filepath.Join(dir, "1.hcl"), filepath.Join(dir, "1.md")}
	if !slices.Equal(got, want) {
		t.Fatalf("\ngot:  %q\nwant: %q", got, want)
	}
}

func writeTemp(t *testing.T, name, src string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	writeFile(t, path, src)
	return path
}

func writeFile(t *testing.T, path, src string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}
