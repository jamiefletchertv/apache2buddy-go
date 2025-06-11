# Apache2buddy-go

Apache2buddy-go is a Go rewrite of the popular Perl script that analyzes your Apache HTTP Server configuration and provides actionable recommendations for memory optimization and performance tuning.

**⚠️ SECURITY NOTICE: This tool requires root privileges and should only be run on systems you own and control. Review the source code before building and running. Use at your own risk.**

## What does it do?

Running Apache can be tricky to get right. Too few worker processes and you're not utilizing your server's full potential. Too many and you'll run out of memory, causing your server to swap and crawl to a halt.

Apache2buddy-go analyzes your current Apache setup by:
- Examining running Apache processes and their actual memory usage
- Parsing your Apache configuration files
- Checking your system's available memory
- Analyzing Apache error logs for issues
- Providing specific recommendations for MaxRequestWorkers/MaxClients settings

The result? You get concrete numbers on how to tune your Apache configuration for optimal performance without running out of memory.

## Features

- **Memory Analysis**: Real-time analysis of Apache worker memory consumption
- **Configuration Parsing**: Automatic detection and parsing of Apache config files
- **Multiple MPM Support**: Works with prefork, worker, and event MPMs
- **mod_status Integration**: Enhanced analysis when mod_status is available
- **Log Analysis**: Scans Apache error logs for MaxClients exceeded warnings
- **Service Detection**: Accounts for memory used by MySQL, PHP-FPM, Redis, and other services
- **Historical Logging**: Tracks recommendations over time
- **Debug Mode**: Detailed troubleshooting output for complex setups
- **Exit Codes**: Scriptable with meaningful exit codes (0=OK, 1=Warning, 2=Critical)

## Installation

**apache2buddy-go is distributed as source code only. No pre-compiled binaries are provided.**

### Build from Source (Required)

```bash
git clone https://github.com/jamiefletcherdev/apache2buddy-go.git
cd apache2buddy-go
go build -o apache2buddy-go .
```

### Alternative build methods

```bash
# Using Make
make build

# With version information
go build -ldflags="-s -w" -o apache2buddy-go .
```

## Usage

``apache2buddy-go`` must be run as root to access process memory information and Apache configuration files.

### Basic Usage

```bash
sudo ./apache2buddy-go
```

### Command Line Options

```bash
sudo ./apache2buddy-go [OPTIONS]

Options:
  -debug         Enable debug mode for detailed troubleshooting
  -help          Show help information
  -version       Show version information
  -history N     Show last N entries from apache2buddy-go log file
```

### Examples

```bash
# Basic analysis
sudo ./apache2buddy-go

# Debug mode with detailed output
sudo ./apache2buddy-go -debug

# View historical recommendations
sudo ./apache2buddy-go -history 10
```

## Sample Output

```
Apache2buddy Go
==================================

Server Version: Apache 2.4.41
Server MPM: prefork
Server Built: Oct  6 2020 16:28:31

Total RAM: 2048 MB
Available RAM: 1456 MB

Current MaxRequestWorkers: 256

Apache processes found: 12
Memory usage per process: 18.5 MB (smallest), 24.3 MB (average), 31.2 MB (largest)

Current memory usage: 399.4 MB (27.4% of available)
Recommended MaxRequestWorkers: 46
Projected memory usage: 1435.2 MB (98.6% of available)

⚠️  RESULT: Your Apache configuration could be improved.
Consider reducing MaxRequestWorkers to 46 to prevent memory issues.

Configuration file: /etc/apache2/apache2.conf

To implement changes, edit your Apache configuration:
<IfModule mpm_prefork_module>
    MaxRequestWorkers 46
</IfModule>

Then restart Apache to apply changes.
```

## Requirements

- **Root Access**: Must be run as root to access process information
- **Apache HTTP Server**: Apache must be running
- **Go 1.19+**: For building from source
- **System Commands**: Requires `ps` and `pmap` commands

### Supported Operating Systems

- Linux (all distributions)
- FreeBSD
- macOS (for development/testing)

### Supported Apache Versions

- Apache 2.2.x
- Apache 2.4.x

## Understanding the Results

### Exit Codes

- **0 (OK)**: Configuration looks good
- **1 (WARNING)**: Configuration could be improved
- **2 (CRITICAL)**: Configuration needs immediate attention

### Status Messages

- **OK**: Your current MaxRequestWorkers setting is within safe limits
- **WARNING**: You're close to memory limits or could optimize further
- **CRITICAL**: Your current setting will likely cause memory issues

### Memory Calculations

Apache2buddy-go uses the largest Apache process memory footprint for calculations to ensure conservative recommendations. This prevents out-of-memory situations when processes grow under load.

## Configuration Examples

### Prefork MPM

```apache
<IfModule mpm_prefork_module>
    StartServers          8
    MinSpareServers       5
    MaxSpareServers      20
    MaxRequestWorkers    46
    MaxConnectionsPerChild 10000
</IfModule>
```

### Worker MPM

```apache
<IfModule mpm_worker_module>
    StartServers          2
    MinSpareThreads      25
    MaxSpareThreads      75
    ThreadsPerChild      25
    MaxRequestWorkers    150
    MaxConnectionsPerChild 10000
</IfModule>
```

## Troubleshooting

### "This script must be run as root"

apache2buddy-go needs root privileges to:
- Access process memory information via /proc
- Read Apache configuration files
- Analyze Apache error logs

### "No Apache worker processes found"

This usually means:
- Apache is not running
- Apache processes are running under a different name
- Use `-debug` flag to see what processes are detected

### "Apache config file not found"

apache2buddy-go looks for config files in standard locations:
- `/etc/apache2/apache2.conf`
- `/etc/httpd/conf/httpd.conf`
- `/usr/local/apache2/conf/httpd.conf`

Use `-debug` to see which paths are being checked.

### "Could not get Apache status info"

Enable mod_status for enhanced analysis:

```apache
<Location "/server-status">
    SetHandler server-status
    Require local
</Location>
ExtendedStatus On
```

## Historical Data

apache2buddy-go logs all analysis results to `/var/log/apache2buddy-go.log` for tracking changes over time:

```bash
# View recent entries
sudo ./apache2buddy-go -history 5

# View full log
sudo cat /var/log/apache2buddy-go.log
```

## Differences from Original Perl Version

This Go implementation includes several enhancements:

- **Better Error Handling**: More robust parsing and error recovery
- **Enhanced Service Detection**: Improved detection of PHP-FPM, MySQL, Redis
- **Timeout Protection**: Network and file operations have timeouts
- **Structured Logging**: JSON-compatible log format
- **Extended Debug Mode**: More detailed troubleshooting information
- **Cross-Platform**: Better support for different Linux distributions

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Credits

Based on the original [apache2buddy.pl](https://github.com/richardforth/apache2buddy) by Richard Forth. This Go implementation was created to provide better performance, error handling, and cross-platform compatibility.

## Support

- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/jamiefletcherdev/apache2buddy-go/issues)
- **Documentation**: Check the [Wiki](https://github.com/jamiefletcherdev/apache2buddy-go/wiki) for additional guides
- **Community**: Join discussions in [GitHub Discussions](https://github.com/jamiefletcherdev/apache2buddy-go/discussions)