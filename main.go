// gosq-codegen introspects a database schema and generates type-safe Go
// table and field definitions for use with github.com/libliflin/gosq.
//
// Usage:
//
//	gosq-codegen -dsn "postgres://user:pass@localhost:5432/mydb" -out schema/
//
// The generated code uses gosq's NewTable and NewField constructors, giving
// you compile-time column references instead of raw strings. The output looks
// like:
//
//	package schema
//
//	import . "github.com/libliflin/gosq"
//
//	var Users = NewTable("users")
//	var (
//	    UsersID    = NewField("users.id")
//	    UsersName  = NewField("users.name")
//	    UsersEmail = NewField("users.email")
//	)
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/libliflin/gosq-codegen/internal/codegen"
	"github.com/libliflin/gosq-codegen/internal/introspect"
	_ "github.com/lib/pq"
)

func main() {
	dsn := flag.String("dsn", "", "PostgreSQL connection string (required)")
	out := flag.String("out", "schema/", "output directory")
	pkg := flag.String("pkg", "schema", "Go package name for generated file")
	schema := flag.String("schema", "public", "PostgreSQL schema to introspect")
	dotImport := flag.Bool("dot-import", true, "use dot-import for gosq (import . \"github.com/libliflin/gosq\")")
	flag.Parse()

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "gosq-codegen: -dsn is required")
		flag.Usage()
		os.Exit(1)
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gosq-codegen: open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tables, err := introspect.Tables(ctx, db, *schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gosq-codegen: introspect: %v\n", err)
		os.Exit(1)
	}

	if len(tables) == 0 {
		fmt.Fprintf(os.Stderr, "gosq-codegen: warning: no tables found in schema %q\n", *schema)
	}

	src, err := codegen.Generate(tables, codegen.Config{Package: *pkg, DotImport: *dotImport})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gosq-codegen: generate: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*out, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "gosq-codegen: mkdir %s: %v\n", *out, err)
		os.Exit(1)
	}

	outFile := filepath.Join(*out, *pkg+".go")
	if err := os.WriteFile(outFile, src, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "gosq-codegen: write %s: %v\n", outFile, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "wrote %s\n", outFile)
}
