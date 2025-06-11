# Contributing to Apache2buddy-go

Thanks for your interest in contributing! Apache2buddy-go is a tool used by system administrators worldwide, so your contributions help make server management easier for everyone.

## Getting Started

### Development Environment

You'll need:
- Go 1.19 or later
- A Linux system for testing (VMs work fine)
- Apache HTTP Server installed for integration testing
- Basic understanding of Apache configuration

### Setting Up

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/yourusername/apache2buddy-go.git
   cd apache2buddy-go
   ```
3. Build and test:
   ```bash
   go build -o apache2buddy-go .
   sudo ./apache2buddy-go -debug
   ```

## Types of Contributions

### Bug Reports

Before filing a bug report, please:
- Check existing issues to avoid duplicates
- Test with the latest version
- Run with `-debug` flag to gather detailed information

When reporting bugs, include:
- Your operating system and version
- Apache version (`apache2 -v` or `httpd -v`)
- Complete error output with `-debug` flag
- Your Apache configuration (sanitized if needed)
- Steps to reproduce the issue

### Feature Requests

We welcome feature requests! Good requests include:
- A clear description of the problem you're trying to solve
- Why existing functionality doesn't work for your use case
- Examples of how you'd expect it to work
- Whether you're willing to help implement it

### Code Contributions

#### What We're Looking For

- **Bug fixes**: Always welcome, especially with test cases
- **Apache compatibility**: Support for newer Apache versions or distributions
- **Performance improvements**: Better memory analysis or faster execution
- **Error handling**: More robust parsing or better error messages
- **Documentation**: Code comments, examples, or README improvements

#### What We're Less Interested In

- Large architectural changes without discussion first
- Features that significantly increase complexity
- Platform-specific code that doesn't benefit the majority of users

## Development Guidelines

### Code Style

We follow standard Go conventions:
- Use `gofmt` to format your code
- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Write clear, descriptive variable and function names
- Keep functions focused and reasonably sized

### Project Structure

```
apache2buddy-go/
├── main.go              # Main entry point and CLI handling
├── internal/
│   ├── analysis/        # Memory calculations and recommendations
│   ├── config/          # Apache configuration parsing
│   ├── debug/           # Debug logging and tracing
│   ├── logs/            # Apache log analysis
│   ├── output/          # Report formatting and display
│   ├── process/         # Process discovery and memory analysis
│   ├── status/          # mod_status integration
│   └── system/          # System information and service detection
└── examples/            # Test data and examples
```

### Testing

Before submitting:

1. **Build successfully**: `go build .`
2. **Test on real systems**: Run against actual Apache setups
3. **Test edge cases**: Try with missing config files, no Apache running, etc.
4. **Debug mode**: Ensure `-debug` output is helpful
5. **Different Apache versions**: Test on Apache 2.2 and 2.4 if possible

### Debug Information

When working on the code, use the debug package extensively:
```go
debug.Info("Starting configuration parse")
defer debug.Trace("parseConfigFile")()
debug.DumpStruct("ParsedConfig", config)
```

This helps with troubleshooting and makes the codebase more maintainable.

## Submitting Changes

### Pull Request Process

1. **Create a feature branch**: `git checkout -b fix-memory-calculation`
2. **Make your changes**: Keep commits focused and logical
3. **Test thoroughly**: Especially on different systems if possible
4. **Update documentation**: README, code comments, etc.
5. **Submit pull request**: Include a clear description

### Pull Request Guidelines

**Good PR titles:**
- "Fix memory calculation for Alpine Linux"
- "Add support for Apache 2.4.50+ configuration syntax"
- "Improve error handling when mod_status is disabled"

**Bad PR titles:**
- "Update code"
- "Fix bug"
- "Changes"

**PR Description should include:**
- What problem this solves
- How you tested it
- Any breaking changes
- Screenshots/output examples if relevant

### Commit Messages

Write clear commit messages:
```
Fix memory calculation on systems with large processes

The previous calculation could overflow on systems with processes
using more than 2GB of memory. This switches to int64 for memory
values and adds bounds checking.

Fixes #123
```

## Community Guidelines

### Be Respectful

This is a technical project, but remember there are humans behind every contribution. Be patient with newcomers and constructive in feedback.

### Focus on the User

Remember that system administrators rely on this tool in production environments. Stability and reliability are more important than new features.

### Ask Questions

If you're unsure about anything, just ask! Open an issue for discussion before spending time on large changes.

## Getting Help

- **Questions about contributing**: Open a GitHub issue with the "question" label
- **Technical discussions**: Use GitHub Discussions
- **Quick questions**: Comments on existing issues or PRs are fine

## Recognition

Contributors are recognized in the project's commit history and GitHub contributors list. Significant contributions may be mentioned in release notes.

Thanks for helping make Apache2buddy-go better!