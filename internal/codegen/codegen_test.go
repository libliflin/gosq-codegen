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

func TestGenerateDotImportFalse(t *testing.T) {
	tables := []introspect.Table{
		{
			Schema: "public",
			Name:   "users",
			Columns: []introspect.Column{
				{Name: "id", DataType: "integer", IsNullable: false, OrdinalPos: 1},
			},
		},
	}

	got, err := Generate(tables, Config{Package: "schema", DotImport: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "package schema\n\nimport \"github.com/libliflin/gosq\"\n\nvar Users = gosq.NewTable(\"users\")\n\nvar (\n\tUsersID = gosq.NewField(\"users.id\")\n)\n"
	if string(got) != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenerateDigitLeadingColumn(t *testing.T) {
	tables := []introspect.Table{
		{
			Schema: "public",
			Name:   "accounts",
			Columns: []introspect.Column{
				{Name: "2fa_enabled", DataType: "boolean", IsNullable: false, OrdinalPos: 1},
			},
		},
	}

	got, err := Generate(tables, Config{Package: "schema", DotImport: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "package schema\n\nimport . \"github.com/libliflin/gosq\"\n\nvar Accounts = NewTable(\"accounts\")\n\nvar (\n\tAccounts_2faEnabled = NewField(\"accounts.2fa_enabled\")\n)\n"
	if string(got) != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenerateAllUnderscoreColumn(t *testing.T) {
	tables := []introspect.Table{
		{
			Schema: "public",
			Name:   "items",
			Columns: []introspect.Column{
				{Name: "___", DataType: "text", IsNullable: false, OrdinalPos: 1},
			},
		},
	}

	got, err := Generate(tables, Config{Package: "schema", DotImport: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "package schema\n\nimport . \"github.com/libliflin/gosq\"\n\nvar Items = NewTable(\"items\")\n\nvar (\n\tItems_ = NewField(\"items.___\")\n)\n"
	if string(got) != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, want)
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
