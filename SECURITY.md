# Security Policy

## Important Notice

**apache2buddy-go is provided "AS IS" without warranty of any kind.** Users are responsible for reviewing the source code and running this software at their own risk.

This tool requires root privileges and analyzes system internals. **You should only run apache2buddy-go on systems you own and control.**

## Use at Your Own Risk

By using apache2buddy-go, you acknowledge that:

- This software requires root/administrator privileges
- You have reviewed the source code before compilation
- You understand the security implications of running system analysis tools
- You accept full responsibility for any consequences of running this software
- No warranties or guarantees are provided

## Source Code Only

**apache2buddy-go is distributed as source code only.** 

- No pre-compiled binaries are provided
- Users must clone the repository and build from source
- This ensures you can review the code before execution
- Building from source is the only supported installation method

## Security Best Practices

When using apache2buddy-go:

1. **Review the code**: Always examine the source before building
2. **Build from source**: Clone the official repository and compile yourself
3. **Controlled environments**: Only run on systems you own and control
4. **Understand the analysis**: Know what system information is being accessed
5. **Monitor execution**: Use debug mode to see exactly what the tool is doing

## What apache2buddy-go Does

This tool performs **read-only** analysis:
- Reads process memory information from `/proc`
- Parses Apache configuration files
- Analyzes Apache error logs
- Checks system memory and running processes
- Creates local log entries for historical tracking

**It does NOT:**
- Modify any system files or configurations
- Make network connections
- Install or change software
- Transmit data externally
- Store sensitive information

## Reporting Issues

If you discover potential security concerns:

1. Review the source code to confirm the issue
2. Open a GitHub issue with:
   - Clear description of the concern
   - Relevant code references
   - Suggested improvements

## Disclaimer

This project follows the same security model as the original apache2buddy.pl - **use at your own risk with full understanding of what the tool does.** 

System administrators are expected to:
- Understand the tools they run
- Review source code before execution
- Accept responsibility for their system security

**No support is provided for security issues arising from misuse or lack of understanding of the tool's functionality.**