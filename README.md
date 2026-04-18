# dotgo - Modern Dotfiles Management

A modern, Go-based CLI tool for managing dotfiles with package-based organization, intelligent symlink management, and migration capabilities.

## Features

- **Package-based Organization**: Organize your dotfiles into logical packages
- **Intelligent Symlink Management**: Automatic symlink creation with conflict handling
- **Profile System**: Support for different environments (home, work, etc.)
- **Template Variables**: Dynamic configuration with template support
- **Migration Tools**: Easy migration from existing dotfiles setups
- **Dependency Management**: Package dependency resolution
- **Homebrew Integration**: Built-in Brewfile support
- **Backup System**: Automatic backup of existing files
- **Dry-run Support**: Preview changes before applying them

## Quick Start

### Installation

```bash
# Build from source
cd dotgo
make build

# Install to GOPATH/bin
make install
```

### Initialize a New Repository

```bash
# Initialize in current directory
dotgo init

# Initialize in specific directory
dotgo init ~/my-dotfiles
```

### Migrate Existing Dotfiles

```bash
# Analyze existing structure
dotgo migrate --analyze-only

# Perform migration
dotgo migrate
```

### Install Packages

```bash
# Install all packages from default profile
dotgo install

# Install specific packages
dotgo install zsh git vim

# Dry run to preview changes
dotgo install --dry-run
```

## Directory Structure

```
.
├── .dotgo/
│   ├── config.yaml          # Main configuration
│   └── backups/             # File backups
├── packages/                # Package definitions
│   ├── zsh/
│   │   ├── package.yaml     # Package configuration
│   │   └── .zshrc          # Dotfile content
│   └── git/
│       ├── package.yaml
│       ├── .gitconfig
│       └── .gitignore
├── profiles/                # Profile configurations
├── templates/               # Template files
└── legacy-install.sh        # Preserved original script
```

## Package Configuration

Each package has a `package.yaml` file:

```yaml
name: zsh
description: Zsh shell configuration
dependencies:
  - homebrew
files:
  - source: .zshrc
    target: ~/.zshrc
  - source: .zsh_aliases
    target: ~/.zsh_aliases
commands:
  pre_install:
    - echo "Installing zsh configuration"
  post_install:
    - echo "Restart your shell or run: source ~/.zshrc"
```

## Main Configuration

The `.dotgo/config.yaml` file contains global settings:

```yaml
version: "1.0"
repository:
  type: local

profiles:
  default:
    name: default
    description: Default profile
    packages:
      - zsh
      - git
      - vim
    variables:
      email: user@example.com
      editor: vim

settings:
  default_profile: default
  backup_dir: .dotgo/backups
  symlink_mode: auto
  conflict_mode: ask
  packages_dir: packages
  profiles_dir: profiles
  templates_dir: templates
```

## Commands

### Core Commands

- `dotgo init [directory]` - Initialize a new dotgo repository
- `dotgo install [packages...]` - Install dotfiles packages
- `dotgo status` - Show current dotfiles status
- `dotgo migrate` - Migrate existing dotfiles

### Package Management

- `dotgo packages list` - List all available packages
- `dotgo packages status` - Show package installation status
- `dotgo packages install <package>` - Install specific packages
- `dotgo packages remove <package>` - Remove specific packages
- `dotgo packages info <package>` - Show package details

### Global Flags

- `--verbose, -v` - Verbose output
- `--dry-run` - Show what would be done without making changes
- `--config` - Specify config file path

## Migration from Existing Dotfiles

dotgo can automatically analyze and migrate your existing dotfiles:

1. **Analysis**: `dotgo migrate --analyze-only`
   - Discovers dotfiles and config directories
   - Suggests package organization
   - Identifies files that need backup

2. **Migration**: `dotgo migrate`
   - Creates package structure
   - Generates configurations
   - Preserves original install scripts
   - Maintains backward compatibility

## Profiles

Profiles allow you to have different configurations for different environments:

```yaml
profiles:
  home:
    packages: [zsh, git, vim, personal]
  work:
    packages: [zsh, git, vim, work-tools]
    variables:
      git_email: work@company.com
```

Use profiles with:
```bash
dotgo install --profile work
```

## Template System

Files can use Go template syntax with profile variables:

```bash
# In .gitconfig template
[user]
    email = {{ .email }}
    name = {{ .name }}
```

## Dependency Management

Packages can depend on other packages:

```yaml
name: development
dependencies:
  - zsh
  - git
  - homebrew
```

Dependencies are automatically resolved and installed in the correct order.

## Backup and Recovery

dotgo automatically backs up existing files before creating symlinks:

- Backups are stored in `.dotgo/backups/`
- Timestamped filenames prevent conflicts
- Use `--restore-backup` when removing packages to restore originals

## Compatibility

dotgo maintains compatibility with traditional dotfiles setups:

- Preserves existing install scripts as reference
- Can run alongside existing configurations
- Migration is non-destructive by default
- Supports gradual adoption

## Development

### Building

```bash
# Build binary
make build

# Build for development (with race detector)  
make dev

# Run tests
make test

# Cross-compile for multiple platforms
make cross-compile

# Create release
make release
```

### Project Structure

```
dotgo/
├── cmd/                     # CLI commands
│   ├── root.go             # Root command
│   ├── init.go             # Init command
│   ├── install.go          # Install command
│   ├── packages.go         # Package management
│   ├── status.go           # Status command
│   └── migrate.go          # Migration command
├── pkg/                     # Core packages
│   ├── config/             # Configuration management
│   ├── packages/           # Package operations
│   ├── symlink/            # Symlink management
│   └── migration/          # Migration tools
├── internal/               # Internal utilities
└── main.go                 # Main entry point
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Comparison with Other Tools

| Feature | dotgo | stow | chezmoi | yadm |
|---------|-------|------|---------|------|
| Package organization | ✅ | ✅ | ❌ | ❌ |
| Migration tools | ✅ | ❌ | ✅ | ❌ |
| Template support | ✅ | ❌ | ✅ | ✅ |
| Dependency management | ✅ | ❌ | ❌ | ❌ |
| Profile system | ✅ | ❌ | ✅ | ❌ |
| Backup system | ✅ | ❌ | ✅ | ❌ |
| Cross-platform | ✅ | ✅ | ✅ | Linux/macOS |
| Language | Go | Perl | Go | Bash |

dotgo combines the best features of existing tools while adding unique capabilities like migration tools, dependency management, and package-based organization.