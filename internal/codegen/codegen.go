// Package codegen renders Go source files from introspected schema metadata.
// The output uses gosq's NewTable and NewField constructors.
package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"

	"github.com/libliflin/gosq-codegen/internal/introspect"
)

// Config controls code generation output.
type Config struct {
	// Package is the Go package name for generated files (default: "schema").
	Package string

	// DotImport controls whether to use dot-import for gosq.
	// When true:  import . "github.com/libliflin/gosq"
	// When false: import "github.com/libliflin/gosq"
	DotImport bool
}

// Generate produces Go source code for the given tables.
func Generate(tables []introspect.Table, cfg Config) ([]byte, error) {
	if cfg.Package == "" {
		cfg.Package = "schema"
	}

	sorted := make([]introspect.Table, len(tables))
	copy(sorted, tables)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "package %s\n\n", cfg.Package)

	if len(sorted) > 0 {
		if cfg.DotImport {
			fmt.Fprintf(&buf, "import . \"github.com/libliflin/gosq\"\n\n")
		} else {
			fmt.Fprintf(&buf, "import \"github.com/libliflin/gosq\"\n\n")
		}
	}

	for _, tbl := range sorted {
		tableIdent := toExported(tbl.Name)
		if cfg.DotImport {
			fmt.Fprintf(&buf, "var %s = NewTable(%q)\n\n", tableIdent, tbl.Name)
		} else {
			fmt.Fprintf(&buf, "var %s = gosq.NewTable(%q)\n\n", tableIdent, tbl.Name)
		}

		if len(tbl.Columns) > 0 {
			fmt.Fprintf(&buf, "var (\n")
			for _, col := range tbl.Columns {
				fieldIdent := tableIdent + toExported(col.Name)
				if cfg.DotImport {
					fmt.Fprintf(&buf, "\t%s = NewField(%q)\n", fieldIdent, tbl.Name+"."+col.Name)
				} else {
					fmt.Fprintf(&buf, "\t%s = gosq.NewField(%q)\n", fieldIdent, tbl.Name+"."+col.Name)
				}
			}
			fmt.Fprintf(&buf, ")\n\n")
		}
	}

	src := bytes.TrimRight(buf.Bytes(), "\n")
	src = append(src, '\n')

	formatted, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("formatting generated source: %w", err)
	}
	return formatted, nil
}

// goInitialisms maps lowercase words to their idiomatic Go uppercase form.
var goInitialisms = map[string]string{
	"id":    "ID",
	"url":   "URL",
	"uri":   "URI",
	"http":  "HTTP",
	"https": "HTTPS",
	"sql":   "SQL",
	"api":   "API",
	"uid":   "UID",
	"uuid":  "UUID",
	"ip":    "IP",
	"io":    "IO",
	"cpu":   "CPU",
	"xml":   "XML",
	"json":  "JSON",
	"rpc":   "RPC",
	"tls":   "TLS",
	"ttl":   "TTL",
}

// toExported converts a snake_case database identifier to an exported Go identifier.
func toExported(name string) string {
	parts := strings.Split(name, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if upper, ok := goInitialisms[strings.ToLower(part)]; ok {
			b.WriteString(upper)
		} else {
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}
