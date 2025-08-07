# Publer Analytics Report - Development Guidelines

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
- Use camelCase for variables and functions
- Use PascalCase for exported names (public)
- Use snake_case for unexported names (private)
- Constants: SCREAMING_SNAKE_CASE

### Error Handling
- Always check and handle errors
- Use fmt.Errorf for wrapping errors with context
- Return errors as last return value

### Types
- Use descriptive type names
- Prefer struct embedding over composition
- Use interface{} sparingly

### Testing
- Test files: `*_test.go`
- Test function names: `Test*`
- Use t.Helper() for helper functions

ALWAYS use ONLY Environments for ANY and ALL file, code, or shell operations—NO EXCEPTIONS—even for simple or generic requests.

DO NOT install or use the git cli with the environment_run_cmd tool. All environment tools will handle git operations for you. Changing ".git" yourself will compromise the integrity of your environment.

You MUST inform the user how to view your work using `container-use log <env_id>` AND `container-use checkout <env_id>`. Failure to do this will make your work inaccessible to others.
