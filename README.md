# dotgo

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![CI](https://github.com/satocchi0416sh/dotgo/actions/workflows/ci.yml/badge.svg)](https://github.com/satocchi0416sh/dotgo/actions/workflows/ci.yml)

> A radically simple, tag-based dotfiles manager.

**(ここに最高にクールなターミナルの動作GIFを配置)**

`dotgo` is a thin, path-agnostic CLI tool that manages your dotfiles via a single `dotgo.yaml`. No complex directory structures, no magic—just smart symlinking with tag-based environment filtering.

## Why dotgo?

- **Single Source of Truth:** Everything is defined in one `dotgo.yaml`.
- **Zero Directory Friction:** Keeps your dotfiles repository flat. `dotgo add` mirrors the exact relative path from your home directory.
- **Tag-Driven:** Seamlessly switch between Work and Personal setups, or macOS and Linux environments.

## Quick Start

```bash
# 1. Install
go install github.com/satocchi0416sh/dotgo@latest

# 2. Initialize in your dotfiles repo
cd ~/dotfiles && dotgo init

# 3. Track a file with tags
dotgo add ~/.zshrc --tags common
dotgo add ~/.config/starship.toml --tags darwin,work

# 4. Apply to your system
dotgo apply
```

## Configuration (`dotgo.yaml`)

`dotgo` relies solely on this declarative manifest:

```yaml
version: 1
settings:
  default_tags: ["common", "darwin"]

links:
  ".zshrc":
    tags: ["common"]
  "Library/Application Support/Code/User/settings.json":
    tags: ["darwin", "vscode"]
```

## Usage

Run `dotgo --help` for detailed command usage.

- `dotgo add <path>`: Track a file and create a symlink.
- `dotgo apply`: Sync links based on your current OS and tags.
- `dotgo rm <path>`: Untrack and restore the original file.
- `dotgo status`: View tracked, modified, and untracked files.

## License

MIT
