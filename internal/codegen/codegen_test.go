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
	want := "package schema\n"
	if string(out) != want {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", string(out), want)
	}
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

	got, err := Generate(tables, Config{Package: "schema", DotImport: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "package schema\n\nimport . \"github.com/libliflin/gosq\"\n\nvar Users = NewTable(\"users\")\n\nvar (\n\tUsersID   = NewField(\"users.id\")\n\tUsersName = NewField(\"users.name\")\n)\n"
	if string(got) != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}
