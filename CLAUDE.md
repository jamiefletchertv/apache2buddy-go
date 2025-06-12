# Apache2buddy-Go Development Guide

## Project Overview
Apache2buddy-go is a Go-based rewrite of the popular apache2buddy Perl script. It analyzes Apache HTTP Server configurations and provides optimization recommendations for memory usage, MPM settings, and performance tuning.

## Architecture & Design Principles

### Core Philosophy
- **Single-purpose tool**: Analyze Apache configurations and provide actionable recommendations
- **Zero external dependencies**: Uses only Go standard library for maximum portability
- **Static binary distribution**: Self-contained executable with no runtime dependencies
- **Cross-platform compatibility**: Supports Linux, macOS, FreeBSD with focus on server environments
- **Local execution**: Designed to run on the same server as Apache for accurate analysis

### Key Components
- **Analysis Engine** (`internal/analysis/`): Memory calculations and optimization recommendations
- **Configuration Parser** (`internal/config/`): Apache configuration file parsing and validation
- **Status Integration** (`internal/status/`): mod_status data collection and processing
- **System Information** (`internal/system/`): Server resource detection and analysis
- **Output Formatting** (`internal/output/`): Report generation and display formatting

## Development Standards

### Code Quality Requirements
- **100% test coverage** for all internal packages
- **Table-driven tests** for comprehensive scenario coverage
- **Benchmark tests** for performance-critical functions
- **Static analysis**: All code must pass `go vet`, `golangci-lint`, and `go fmt`
- **No external dependencies**: Only Go standard library permitted
- **Cross-platform builds**: Must compile for Linux/macOS/FreeBSD on amd64/arm64

### Testing Strategy
1. **Unit Tests**: Complete coverage of all internal packages with edge cases
2. **Integration Tests**: Docker-based testing across multiple Apache distributions
3. **Performance Tests**: Benchmarks for memory calculations and parsing operations
4. **Cross-platform Tests**: Build verification across supported platforms

### Supported Apache Distributions
- **Alpine Linux**: httpd package (primary testing platform)
- **Ubuntu/Debian**: apache2 package with different path conventions
- **CentOS Stream**: httpd package with RHEL ecosystem
- **Rocky Linux**: Enterprise Linux compatible with prefork MPM testing
- **AlmaLinux**: Enterprise Linux compatible with worker MPM testing

## Build & Test Infrastructure

### Makefile Targets
```bash
# Core development
make build          # Build binary
make test           # Run unit tests
make check          # Run all quality checks (fmt, vet, lint, test)

# Testing
make test-unit      # Unit tests only
make test-race      # Race condition detection
make test-cover     # Coverage analysis
make test-integration # Docker-based integration tests
make test-all       # Complete test suite

# Docker integration testing
make docker-test    # Full integration pipeline
make docker-build-containers # Build all Apache containers
make docker-integration-tests # Run apache2buddy-go in each container
make docker-logs    # Show container logs
make docker-status  # Show container status
```

### Integration Test Architecture
Tests run apache2buddy-go **inside** Apache containers (not externally) for authentic environment testing:

```yaml
# docker-compose.test.yml structure
services:
  apache-httpd-test:    # Alpine/httpd (port 8080)
  apache-ubuntu-test:   # Ubuntu/apache2 (port 8081)  
  apache-centos-test:   # CentOS Stream/httpd (port 8082)
  apache-rocky-test:    # Rocky Linux/httpd (port 8083)
  apache-alma-test:     # AlmaLinux/httpd (port 8084)
```

Each container:
- Builds apache2buddy-go from source during container build
- Configures Apache with mod_status enabled
- Runs health checks to ensure proper startup
- Executes apache2buddy-go directly inside the container environment

## Configuration Testing Scenarios

### MPM Configurations
- **Prefork**: Traditional process-based model (Rocky Linux container)
- **Worker**: Hybrid thread/process model (AlmaLinux container)  
- **Event**: Asynchronous event-driven model (default on most platforms)

### Resource Constraints Testing
- Low memory scenarios (512MB)
- High traffic scenarios (1000+ connections)
- Mixed workload patterns

### Apache Configuration Variants
- Default installations
- Custom module configurations
- Virtual host scenarios
- SSL/TLS configurations

## Error Handling & Edge Cases

### Common Issues Addressed
- **Missing mod_status**: Graceful degradation when server-status unavailable
- **Permission issues**: Proper handling of restricted Apache directories
- **Configuration parsing**: Robust handling of malformed Apache configs
- **Memory calculations**: Safe arithmetic with boundary condition checks
- **Network timeouts**: Resilient HTTP client for mod_status requests

### Historical Fixes Applied
- Module naming consistency (`apache2buddy` → `apache2buddy-go`)
- Docker health check tool selection (wget vs curl per distribution)
- Service naming in docker-compose and Makefile alignment
- Apache module dependencies (log_config_module, unixd_module)

## CI/CD Pipeline

### GitHub Actions Workflow
```yaml
# Planned pipeline stages
- Code quality checks (lint, vet, fmt)
- Unit test execution with coverage reporting
- Cross-platform build verification
- Docker integration test suite
- Security scanning (gosec)
- Performance regression testing
- Release artifact generation
```

### Quality Gates
- All tests must pass (unit + integration)
- Code coverage > 95%
- No linting violations
- Successful builds on all target platforms
- Docker integration tests pass on all distributions

## Development Workflow

### Adding New Features
1. Write comprehensive unit tests first (TDD approach)
2. Implement feature following existing patterns
3. Add integration test scenarios if applicable
4. Update documentation and help text
5. Verify cross-platform compatibility
6. Run complete test suite (`make test-all`)

### Debugging Integration Tests
```bash
# Start containers individually
make docker-up
make docker-status

# Check specific container logs
docker-compose -f docker-compose.test.yml logs apache-httpd-test

# Execute commands in containers
docker exec -it apache2buddy-test-httpd sh
docker exec apache2buddy-test-ubuntu apache2buddy-go -v

# Clean up after debugging
make docker-down
```

### Performance Optimization
- Use `make benchmarks` for performance regression testing
- Profile memory usage with `go test -memprofile`
- Optimize parsing algorithms for large Apache configurations
- Monitor startup time and memory footprint

## Contributing Guidelines

### Code Style
- Follow Go standard formatting (`go fmt`)
- Use descriptive variable names
- Add comments for complex logic only
- Keep functions focused and testable
- Avoid global state

### Testing Requirements
- Every public function must have unit tests
- Use table-driven tests for multiple scenarios
- Include edge cases and error conditions
- Benchmark performance-critical code paths
- Integration tests for Apache interaction

### Documentation
- Update CLAUDE.md for architectural changes
- Maintain inline code documentation
- Update README.md for user-facing changes
- Document breaking changes in commit messages

## Future Roadmap

### Planned Enhancements
1. **Configuration scenario testing**: Multiple Apache configurations per distribution
2. **GitHub Actions CI/CD**: Automated testing and release pipeline
3. **Performance monitoring**: Benchmark tracking over time
4. **Security scanning**: Integration with gosec and vulnerability databases
5. **Multi-architecture support**: ARM64 and additional platforms

### Technical Debt
- None currently identified (codebase is well-maintained)
- Monitor for dependency drift (currently zero external deps)
- Regular security updates for base Docker images

## Troubleshooting

### Common Development Issues
- **Docker permission errors**: Ensure Docker daemon is running and user has permissions
- **Test timeouts**: Check if Apache containers are starting properly with `make docker-status`
- **Cross-platform build failures**: Verify CGO_ENABLED=0 for static builds
- **Integration test failures**: Check Apache configuration syntax in test configs

### Quick Diagnostics
```bash
# Check overall project health
make check

# Verify all containers build correctly
make docker-build-containers

# Test specific platform
docker exec apache2buddy-test-ubuntu apache2buddy-go -v

# Performance check
make benchmarks
```

This development guide should be referenced for all architectural decisions, testing approaches, and development workflows. The project prioritizes simplicity, reliability, and comprehensive testing over feature complexity.