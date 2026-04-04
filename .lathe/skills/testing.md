# Go Testing Patterns

## Table-Driven Tests

The standard Go testing pattern. Use for any function with multiple input/output cases:

```go
func TestFoo(t *testing.T) {
    tests := []struct {
        name string
        input string
        want  string
    }{
        {"empty", "", ""},
        {"basic", "hello", "HELLO"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Foo(tt.input)
            if got != tt.want {
                t.Errorf("Foo(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

## Golden File Tests

For code generation, use golden files to verify output:

```go
func TestGenerate(t *testing.T) {
    got := Generate(input)
    golden := filepath.Join("testdata", t.Name()+".golden")

    if *update {
        os.WriteFile(golden, got, 0644)
        return
    }

    want, _ := os.ReadFile(golden)
    if !bytes.Equal(got, want) {
        t.Errorf("output mismatch; run with -update to regenerate")
    }
}
```

## Test Commands

```bash
go test ./...                    # Run all tests
go test ./... -v                 # Verbose output
go test ./... -run TestFoo       # Run specific test
go test ./... -cover             # Show coverage
go test ./... -race              # Race condition detection
go test ./... -count=1           # Disable test caching
```
