package entryfile

import (
	"bytes"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sreallymatt/changeloggy/internal/changes"
)

// parseHCL reads entries from `change` blocks, labelled with the entry type:
//
//	change "bug" {
//	  body = "fix a crash when the config file is empty"
//	}
func parseHCL(filename string, src []byte) (*changes.Entries, error) {
	entries := &changes.Entries{}
	if err := hclsimple.Decode(filename, src, nil, entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func writeHCL(e *changes.Entries) ([]byte, error) {
	f := hclwrite.NewFile()
	gohcl.EncodeIntoBody(e.WriteEntries(), f.Body())
	return bytes.TrimLeft(hclwrite.Format(f.Bytes()), "\n"), nil
}
