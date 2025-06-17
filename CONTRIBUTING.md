# 🤝 Contributing to GoCat

First off, thank you for considering contributing to GoCat! It's people like you that make GoCat such a great tool. 🎉

## 📋 Table of Contents

- [Code of Conduct](#-code-of-conduct)
- [Getting Started](#-getting-started)
- [Development Setup](#-development-setup)
- [How to Contribute](#-how-to-contribute)
- [Pull Request Process](#-pull-request-process)
- [Coding Standards](#-coding-standards)
- [Testing Guidelines](#-testing-guidelines)
- [Documentation](#-documentation)
- [Community](#-community)

---

## 📜 Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [conduct@gocat.dev](mailto:conduct@gocat.dev).

---

## 🚀 Getting Started

### 🔍 Ways to Contribute

There are many ways to contribute to GoCat:

- 🐛 **Bug Reports**: Found a bug? Let us know!
- 💡 **Feature Requests**: Have an idea? We'd love to hear it!
- 📝 **Documentation**: Help improve our docs
- 🧪 **Testing**: Help us test new features
- 💻 **Code**: Submit patches and new features
- 🌍 **Translation**: Help translate GoCat to other languages
- 📢 **Advocacy**: Tell others about GoCat

### 🎯 Good First Issues

Looking for a place to start? Check out issues labeled with:
- `good first issue` - Perfect for newcomers
- `help wanted` - We need your expertise
- `documentation` - Help improve our docs
- `bug` - Fix existing issues

---

## 🛠️ Development Setup

### 📋 Prerequisites

- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Git**: [Install Git](https://git-scm.com/downloads)
- **Make**: Usually pre-installed on Unix systems
- **Docker** (optional): [Install Docker](https://docs.docker.com/get-docker/)

### 🏗️ Setup Instructions

1. **Fork the repository**
   ```bash
   # Click the "Fork" button on GitHub, then:
   git clone https://github.com/YOUR_USERNAME/gocat.git
   cd gocat
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/ibrahmsql/gocat.git
   ```

3. **Install dependencies**
   ```bash
   make deps
   ```

4. **Verify setup**
   ```bash
   make test
   make build
   ```

5. **Run the application**
   ```bash
   ./build/gocat --help
   ```

### 🔧 Development Tools

We recommend these tools for development:

- **IDE**: VS Code with Go extension, GoLand, or Vim/Neovim with vim-go
- **Linter**: golangci-lint (installed via `make deps`)
- **Formatter**: gofmt (built into Go)
- **Debugger**: Delve (`go install github.com/go-delve/delve/cmd/dlv@latest`)

---

## 🎯 How to Contribute

### 🐛 Reporting Bugs

Before creating a bug report, please:

1. **Search existing issues** to avoid duplicates
2. **Use the latest version** to ensure the bug still exists
3. **Gather information** about your environment

When creating a bug report:

1. Use our [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml)
2. Provide a **clear title** and **detailed description**
3. Include **steps to reproduce** the issue
4. Add **system information** (OS, architecture, Go version)
5. Include **log output** if applicable
6. Add **screenshots** if relevant

### 💡 Suggesting Features

We love new ideas! When suggesting a feature:

1. Use our [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml)
2. **Describe the problem** you're trying to solve
3. **Explain your proposed solution** in detail
4. **Consider alternatives** and explain why your solution is best
5. **Provide examples** of how the feature would be used

### 💻 Contributing Code

#### 🌿 Branching Strategy

We use a simplified Git flow:

- `main` - Stable release branch
- `develop` - Development branch (default)
- `feature/*` - Feature branches
- `bugfix/*` - Bug fix branches
- `hotfix/*` - Critical fixes for production

#### 🔄 Workflow

1. **Create a branch**
   ```bash
   git checkout develop
   git pull upstream develop
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write clean, readable code
   - Follow our coding standards
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**
   ```bash
   make test
   make lint
   make check
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add amazing new feature"
   ```

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   # Then create a Pull Request on GitHub
   ```

---

## 🔄 Pull Request Process

### 📝 Before Submitting

- [ ] **Read the contributing guidelines** (this document)
- [ ] **Search existing PRs** to avoid duplicates
- [ ] **Create an issue first** for major changes
- [ ] **Test your changes** thoroughly
- [ ] **Update documentation** if needed
- [ ] **Follow commit message conventions**

### 📋 PR Checklist

Your PR should:

- [ ] **Have a clear title** describing the change
- [ ] **Reference related issues** (e.g., "Fixes #123")
- [ ] **Include tests** for new functionality
- [ ] **Pass all CI checks**
- [ ] **Update documentation** if needed
- [ ] **Follow coding standards**
- [ ] **Be focused** - one feature/fix per PR

### 🔍 Review Process

1. **Automated checks** run first (CI, tests, linting)
2. **Code review** by maintainers
3. **Feedback incorporation** if needed
4. **Final approval** and merge

**Review criteria:**
- Code quality and style
- Test coverage
- Documentation completeness
- Performance impact
- Security considerations
- Backward compatibility

---

## 📏 Coding Standards

### 🎨 Code Style

We follow standard Go conventions:

- **Use `gofmt`** for formatting
- **Follow `golint`** recommendations
- **Use meaningful names** for variables and functions
- **Write clear comments** for complex logic
- **Keep functions small** and focused
- **Handle errors properly** - don't ignore them

### 📁 Project Structure

```
gocat/
├── cmd/                 # CLI commands
├── internal/            # Private application code
│   ├── config/         # Configuration handling
│   ├── connection/     # Connection management
│   ├── input/          # Input handling
│   └── logger/         # Logging utilities
├── pkg/                # Public packages
├── docs/               # Documentation
├── scripts/            # Build and utility scripts
├── .github/            # GitHub workflows and templates
└── Makefile           # Build automation
```

### 🏷️ Naming Conventions

- **Packages**: lowercase, single word when possible
- **Files**: lowercase with underscores (e.g., `connection_handler.go`)
- **Functions**: CamelCase, exported functions start with capital
- **Variables**: camelCase for local, CamelCase for exported
- **Constants**: ALL_CAPS with underscores
- **Interfaces**: end with "-er" when possible (e.g., `Handler`, `Reader`)

### 📝 Comments

- **Package comments**: Describe the package purpose
- **Function comments**: Start with function name, describe what it does
- **Complex logic**: Explain the "why", not just the "what"
- **TODO comments**: Include issue number when possible

```go
// Package connection provides network connection management utilities.
package connection

// Handler manages network connections and provides methods for
// establishing, maintaining, and closing connections.
type Handler interface {
    // Connect establishes a connection to the specified address.
    // It returns an error if the connection cannot be established.
    Connect(address string) error
}
```

### 🚨 Error Handling

- **Always handle errors** - don't use `_` to ignore them
- **Wrap errors** with context using `fmt.Errorf`
- **Use custom error types** for specific error conditions
- **Log errors** at appropriate levels

```go
// Good
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
doSomething() // ignoring error
```

---

## 🧪 Testing Guidelines

### 📊 Test Coverage

- **Aim for 80%+ coverage** for new code
- **Test both happy path and error cases**
- **Include edge cases** and boundary conditions
- **Use table-driven tests** for multiple scenarios

### 🏗️ Test Structure

```go
func TestConnectionHandler_Connect(t *testing.T) {
    tests := []struct {
        name    string
        address string
        want    error
    }{
        {
            name:    "valid address",
            address: "localhost:8080",
            want:    nil,
        },
        {
            name:    "invalid address",
            address: "invalid",
            want:    ErrInvalidAddress,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            h := NewHandler()
            got := h.Connect(tt.address)
            if got != tt.want {
                t.Errorf("Connect() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 🎯 Test Types

- **Unit tests**: Test individual functions/methods
- **Integration tests**: Test component interactions
- **End-to-end tests**: Test complete workflows
- **Benchmark tests**: Performance testing

### 🏃 Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test -run TestConnectionHandler_Connect ./internal/connection

# Run benchmarks
make test-bench

# Run tests with race detection
go test -race ./...
```

---

## 📚 Documentation

### 📖 Types of Documentation

- **Code comments**: Inline documentation
- **README**: Project overview and quick start
- **API docs**: Generated from code comments
- **User guides**: Detailed usage instructions
- **Developer docs**: Architecture and design decisions

### ✍️ Writing Guidelines

- **Be clear and concise**
- **Use examples** to illustrate concepts
- **Keep it up to date** with code changes
- **Use proper markdown** formatting
- **Include code snippets** with syntax highlighting

### 🔄 Documentation Updates

When making changes that affect documentation:

1. **Update relevant docs** in the same PR
2. **Test documentation** for accuracy
3. **Check for broken links**
4. **Update examples** if needed

---

## 🌍 Community

### 💬 Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Discord**: Real-time chat and community support
- **Email**: [support@gocat.dev](mailto:support@gocat.dev)

### 🎉 Recognition

We appreciate all contributions! Contributors are recognized:

- **Contributors list** in README
- **Release notes** mention significant contributions
- **Special badges** for regular contributors
- **Maintainer status** for exceptional contributors

### 📅 Community Events

- **Monthly community calls** - First Friday of each month
- **Hackathons** - Quarterly virtual events
- **Conference talks** - We speak at Go conferences
- **Workshops** - Hands-on learning sessions

---

## 🚀 Release Process

### 📋 Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### 🔄 Release Cycle

- **Major releases**: Every 6-12 months
- **Minor releases**: Every 1-2 months
- **Patch releases**: As needed for critical fixes
- **Pre-releases**: Beta versions before major releases

### 📝 Changelog

We maintain a [CHANGELOG.md](CHANGELOG.md) with:

- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security improvements

---

## ❓ FAQ

### 🤔 Common Questions

**Q: How long does it take for PRs to be reviewed?**
A: We aim to review PRs within 48 hours. Complex changes may take longer.

**Q: Can I work on multiple issues at once?**
A: Yes, but we recommend focusing on one at a time for better quality.

**Q: Do I need to sign a CLA?**
A: No, we don't require a Contributor License Agreement.

**Q: How do I become a maintainer?**
A: Regular contributors who demonstrate expertise and commitment may be invited to become maintainers.

**Q: What if my PR is rejected?**
A: Don't worry! We'll provide feedback on how to improve it. Rejection is rare and usually due to scope or timing issues.

---

## 🙏 Thank You!

Thank you for taking the time to contribute to GoCat! Every contribution, no matter how small, makes a difference. Together, we're building something amazing! 🚀

---

## 📞 Need Help?

If you have questions about contributing:

- 📖 Check our [documentation](https://docs.gocat.dev)
- 💬 Join our [Discord community](https://discord.gg/gocat)
- 📧 Email us at [contributors@gocat.dev](mailto:contributors@gocat.dev)
- 🐛 Open an issue with the `question` label

**Happy coding!** 🎉