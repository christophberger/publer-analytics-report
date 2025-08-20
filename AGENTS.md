# Publer Analytics Report - Development Guidelines

## Project Overview

Publer Analytics Report is a CLI tool that reads Publer Analytics CSV files and generates a report in Markdown format from the CSV data.

Find all project details in PROJECT.md.

## Build & Test Commands

- `go build` - Build the project
- `go test` - Run all tests
- `go test -v` - Run tests with verbose output
- `go test -run TestFunctionName` - Run a specific test
- `go test ./...` - Run all tests including subpackages
- `go fmt ./...` - Format all code
- `go vet ./...` - Run static analysis
- `golangci-lint run` - Run linter (if installed)


## Code Style Guidelines

### Imports

- Group imports: standard library, third-party, local packages
- Use absolute imports for local packages: `github.com/christophberger/publer-analytics-report/...`

### Naming Conventions

- Prefer short names but ensure they're clear and descriptive. Rule of thumb: The smaller a function, the shorter the names of parameters and local variables can be
- Use camelCase for unexported names (private)
- Use PascalCase for exported names (public)
- Apply the above to all identifiers (variables, functions, constants, type names, etc.)
- Name interfaces with a name ending in -er. Examples: Reader, Writer, WriteCloser
 

### Error Handling

- Always check and handle errors
- Use fmt.Errorf for wrapping errors with context
- Return errors as last return value


### Types

- Use short but descriptive type names
- Prefer struct embedding over composition
- Use empty interfaces sparingly
- Always use "any" when referring to the empty interface


## Testing

- Test files: `*_test.go`
- Test function names: `Test*`
- Use t.Helper() for helper functions


## Security Considerations

ALWAYS use ONLY Environments for ANY and ALL file, code, or shell operations—NO EXCEPTIONS—even for simple or generic requests.

DO NOT install or use the git cli with the environment_run_cmd tool. All environment tools will handle git operations for you. Changing ".git" yourself will compromise the integrity of your environment.

You MUST inform the user how to view your work using `container-use log <env_id>` AND `container-use checkout <env_id>`. Failure to do this will make your work inaccessible to others.
