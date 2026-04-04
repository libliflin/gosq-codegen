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
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "gosq-codegen: not yet implemented")
	os.Exit(1)
}
