package introspect

import "testing"

func TestTableStructure(t *testing.T) {
	tbl := Table{
		Schema: "public",
		Name:   "users",
		Columns: []Column{
			{Name: "id", DataType: "integer", IsNullable: false, OrdinalPos: 1},
			{Name: "email", DataType: "text", IsNullable: false, OrdinalPos: 2},
			{Name: "bio", DataType: "text", IsNullable: true, OrdinalPos: 3},
		},
	}

	if len(tbl.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(tbl.Columns))
	}
	if tbl.Columns[2].IsNullable != true {
		t.Error("expected bio column to be nullable")
	}
}
