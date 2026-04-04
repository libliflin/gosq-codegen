// Package codegen renders Go source files from introspected schema metadata.
// The output uses gosq's NewTable and NewField constructors.
package codegen

import (
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
	_ = cfg // TODO: implement
	_ = tables
	return nil, nil
}
