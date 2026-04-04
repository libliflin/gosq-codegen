package codegen

import (
	"testing"

	"github.com/libliflin/gosq-codegen/internal/introspect"
)

func TestGenerateEmpty(t *testing.T) {
	out, err := Generate(nil, Config{Package: "schema", DotImport: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// TODO: once Generate is implemented, assert output content
	_ = out
}

func TestGenerateSingleTable(t *testing.T) {
	tables := []introspect.Table{
		{
			Schema: "public",
			Name:   "users",
			Columns: []introspect.Column{
				{Name: "id", DataType: "integer", IsNullable: false, OrdinalPos: 1},
				{Name: "name", DataType: "text", IsNullable: false, OrdinalPos: 2},
			},
		},
	}

	out, err := Generate(tables, Config{Package: "schema", DotImport: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// TODO: assert generated code matches expected output
	_ = out
}
