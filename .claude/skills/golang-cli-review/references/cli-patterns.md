# Go CLI Best Practices Reference

## Table of Contents
1. [CLI Framework Patterns](#cli-framework-patterns)
2. [Error Handling](#error-handling)
3. [Flag and Argument Design](#flag-and-argument-design)
4. [Output and UX](#output-and-ux)
5. [Testing CLI Applications](#testing-cli-applications)
6. [Security Considerations](#security-considerations)
7. [Performance Patterns](#performance-patterns)

---

## CLI Framework Patterns

### Cobra (Recommended)
```go
// Good: Proper command structure
var rootCmd = &cobra.Command{
    Use:   "mytool",
    Short: "A brief description",
    Long:  `A longer description with examples and usage.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Use RunE over Run to return errors
        return nil
    },
}

// Good: Subcommand with proper setup
func NewServeCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:     "serve [flags]",
        Short:   "Start the server",
        Example: "  mytool serve --port 8080",
        RunE:    runServe,
    }
    cmd.Flags().IntP("port", "p", 8080, "port to listen on")
    return cmd
}
```

### Common Anti-patterns
```go
// Bad: Using Run instead of RunE (loses error context)
Run: func(cmd *cobra.Command, args []string) {
    if err := doSomething(); err != nil {
        fmt.Println(err) // Error swallowed
        os.Exit(1)
    }
}

// Bad: Global state for configuration
var globalConfig Config // Avoid

// Good: Pass config through context or command
cmd.SetContext(ctx)
```

---

## Error Handling

### Exit Codes
```go
const (
    ExitSuccess         = 0
    ExitError           = 1
    ExitUsageError      = 2
    ExitConfigError     = 78
    ExitPermissionError = 77
)

// Good: Meaningful exit codes
func main() {
    if err := rootCmd.Execute(); err != nil {
        var usageErr *UsageError
        if errors.As(err, &usageErr) {
            os.Exit(ExitUsageError)
        }
        os.Exit(ExitError)
    }
}
```

### Error Wrapping
```go
// Good: Wrap with context
if err := loadConfig(path); err != nil {
    return fmt.Errorf("loading config from %s: %w", path, err)
}

// Good: Use errors.Is/As for checking
if errors.Is(err, os.ErrNotExist) {
    return fmt.Errorf("config file not found: %s", path)
}
```

### User-Friendly Errors
```go
// Bad: Technical error exposed
return err // "open /foo/bar: permission denied"

// Good: Actionable message
return fmt.Errorf("cannot read config file %q: %w\nTry: chmod 644 %s", path, err, path)
```

---

## Flag and Argument Design

### Flag Naming Conventions
```go
// Good: Consistent naming
cmd.Flags().StringP("output", "o", "", "output file path")
cmd.Flags().BoolP("verbose", "v", false, "enable verbose output")
cmd.Flags().BoolP("quiet", "q", false, "suppress non-error output")
cmd.Flags().Bool("dry-run", false, "show what would be done")

// Bad: Inconsistent
cmd.Flags().String("outputFile", "", "")  // camelCase
cmd.Flags().Bool("DryRun", false, "")     // PascalCase
```

### Required vs Optional
```go
// Good: Mark required flags explicitly
cmd.Flags().StringP("config", "c", "", "config file (required)")
cmd.MarkFlagRequired("config")

// Good: Provide sensible defaults
cmd.Flags().IntP("timeout", "t", 30, "timeout in seconds")
cmd.Flags().StringP("format", "f", "json", "output format (json, yaml, table)")
```

### Validation
```go
// Good: Validate early in PreRunE
PreRunE: func(cmd *cobra.Command, args []string) error {
    format, _ := cmd.Flags().GetString("format")
    if !slices.Contains([]string{"json", "yaml", "table"}, format) {
        return fmt.Errorf("invalid format %q: must be json, yaml, or table", format)
    }
    return nil
},
```

---

## Output and UX

### Structured Output
```go
// Good: Support multiple output formats
type Output struct {
    format string
    writer io.Writer
}

func (o *Output) Print(v any) error {
    switch o.format {
    case "json":
        enc := json.NewEncoder(o.writer)
        enc.SetIndent("", "  ")
        return enc.Encode(v)
    case "yaml":
        return yaml.NewEncoder(o.writer).Encode(v)
    case "table":
        return o.printTable(v)
    }
    return nil
}
```

### Stderr vs Stdout
```go
// Good: Errors and progress to stderr, data to stdout
func run(cmd *cobra.Command, args []string) error {
    // Progress/status to stderr (allows piping stdout)
    fmt.Fprintln(cmd.ErrOrStderr(), "Processing...")

    // Actual output to stdout
    result := process()
    fmt.Fprintln(cmd.OutOrStdout(), result)
    return nil
}
```

### Interactive vs Non-interactive
```go
// Good: Detect TTY and adapt behavior
func isInteractive() bool {
    return term.IsTerminal(int(os.Stdin.Fd()))
}

// Good: Disable prompts in non-interactive mode
if !isInteractive() && !forceFlag {
    return errors.New("refusing to run destructively without --force in non-interactive mode")
}
```

---

## Testing CLI Applications

### Command Testing
```go
func TestServeCommand(t *testing.T) {
    // Good: Test command execution
    cmd := NewServeCmd()
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"--port", "9090"})

    err := cmd.Execute()
    require.NoError(t, err)
    assert.Contains(t, buf.String(), "listening on :9090")
}
```

### Table-Driven Tests
```go
func TestValidateFlags(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr string
    }{
        {"valid format", []string{"--format", "json"}, ""},
        {"invalid format", []string{"--format", "xml"}, "invalid format"},
        {"missing required", []string{}, "required flag"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewCmd()
            cmd.SetArgs(tt.args)
            err := cmd.Execute()
            if tt.wantErr != "" {
                assert.ErrorContains(t, err, tt.wantErr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Testing
```go
func TestCLIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Build binary
    binary := filepath.Join(t.TempDir(), "mytool")
    cmd := exec.Command("go", "build", "-o", binary, "./cmd/mytool")
    require.NoError(t, cmd.Run())

    // Run and verify
    out, err := exec.Command(binary, "version").Output()
    require.NoError(t, err)
    assert.Contains(t, string(out), "v1.0.0")
}
```

---

## Security Considerations

### Input Validation
```go
// Good: Sanitize file paths
func validatePath(p string) error {
    clean := filepath.Clean(p)
    if filepath.IsAbs(p) && !strings.HasPrefix(clean, allowedPrefix) {
        return fmt.Errorf("path %q outside allowed directory", p)
    }
    return nil
}

// Good: Validate URLs
func validateURL(raw string) (*url.URL, error) {
    u, err := url.Parse(raw)
    if err != nil {
        return nil, err
    }
    if u.Scheme != "https" {
        return nil, errors.New("only HTTPS URLs allowed")
    }
    return u, nil
}
```

### Secrets Handling
```go
// Bad: Secrets in flags (visible in ps)
cmd.Flags().String("api-key", "", "API key")

// Good: Environment variables or files
cmd.Flags().String("api-key-file", "", "path to API key file")

// Good: Read from stdin for secrets
func readSecret() (string, error) {
    if term.IsTerminal(int(os.Stdin.Fd())) {
        fmt.Fprint(os.Stderr, "Enter API key: ")
        b, err := term.ReadPassword(int(os.Stdin.Fd()))
        fmt.Fprintln(os.Stderr)
        return string(b), err
    }
    // Non-interactive: read from stdin
    b, err := io.ReadAll(os.Stdin)
    return strings.TrimSpace(string(b)), err
}
```

### Command Injection Prevention
```go
// Bad: Shell injection risk
cmd := exec.Command("sh", "-c", fmt.Sprintf("echo %s", userInput))

// Good: Direct exec, no shell
cmd := exec.Command("echo", userInput)

// Good: Validate before shell use if unavoidable
if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(input) {
    return errors.New("invalid characters in input")
}
```

---

## Performance Patterns

### Lazy Initialization
```go
// Good: Only load what's needed
var configOnce sync.Once
var config *Config

func getConfig() *Config {
    configOnce.Do(func() {
        config = loadConfig()
    })
    return config
}
```

### Context and Cancellation
```go
// Good: Respect context cancellation
func longOperation(ctx context.Context) error {
    for i := 0; i < 100; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            processItem(i)
        }
    }
    return nil
}

// Good: Handle SIGINT/SIGTERM
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(),
        os.Interrupt, syscall.SIGTERM)
    defer cancel()

    if err := rootCmd.ExecuteContext(ctx); err != nil {
        os.Exit(1)
    }
}
```

### Resource Cleanup
```go
// Good: Proper cleanup with defer
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()

    // Process file
    return nil
}

// Good: Cleanup on interrupt
func runServer(ctx context.Context) error {
    srv := &http.Server{Addr: ":8080"}

    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        srv.Shutdown(shutdownCtx)
    }()

    return srv.ListenAndServe()
}
```
