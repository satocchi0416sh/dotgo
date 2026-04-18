# Template System Documentation

## Overview

dotgo supports a powerful template system that allows you to manage configuration files with sensitive or environment-specific data. This is particularly useful for files containing secrets, API keys, personal information, or machine-specific settings that shouldn't be committed to a public repository.

## Key Features

- **Automatic Template Detection**: Files ending with `.tmpl` are automatically recognized as templates
- **Secure Secrets Management**: Keep sensitive data separate from your dotfiles repository
- **Environment Variable Support**: Access system environment variables and custom .env files
- **Git Integration**: Special support for Git include patterns
- **Validation**: Ensures all required secrets are available before processing

## Quick Start

### 1. Creating a Template File

Add `.tmpl` to any configuration file that needs templating:

```bash
# Create a template for your Git configuration
cp ~/.gitconfig ~/.gitconfig.tmpl
```

Edit the template to use template variables:

```gitconfig
# ~/.gitconfig.tmpl
[user]
    name = {{ secret "GIT_USER_NAME" | required }}
    email = {{ secret "GIT_USER_EMAIL" | required }}
    signingkey = {{ secret "GIT_SIGNING_KEY" }}

[commit]
    gpgsign = {{ hasSecret "GIT_SIGNING_KEY" }}

# Include machine-specific configuration
{{ gitInclude "~/.config/git/local" }}
```

### 2. Adding the Template to dotgo

```bash
dotgo add ~/.gitconfig.tmpl --package git
```

dotgo will:

- Move the template to `~/.dotfiles/packages/git/files/gitconfig.tmpl`
- Automatically mark it as a template in the package configuration
- Strip `.tmpl` from the target path (installs as `~/.gitconfig`)

### 3. Setting Up Secrets

Create a `.env` file with your secrets:

```bash
# ~/.config/dotgo/secrets/.env
GIT_USER_NAME="John Doe"
GIT_USER_EMAIL="john@example.com"
GIT_SIGNING_KEY="ABC123DEF456"
```

### 4. Installing the Package

```bash
dotgo install git
```

The template will be processed and installed to `~/.gitconfig` with your secrets injected.

## Template Functions

### Secret Management

| Function                 | Description                          | Example                                        |
| ------------------------ | ------------------------------------ | ---------------------------------------------- |
| `secret "KEY" [default]` | Get secret value, optional default   | `{{ secret "API_KEY" "dev-key" }}`             |
| `hasSecret "KEY"`        | Check if secret exists               | `{{ if hasSecret "SIGNING_KEY" }}...{{ end }}` |
| `envFile "path"`         | Load environment variables from file | `{{ envFile "~/.env.production" }}`            |
| `gitInclude "path"`      | Generate Git include directive       | `{{ gitInclude "~/.gitconfig.local" }}`        |

### Environment Variables

| Function             | Description              | Example                               |
| -------------------- | ------------------------ | ------------------------------------- |
| `env "VAR"`          | Get environment variable | `{{ env "HOME" }}`                    |
| `default value expr` | Provide default value    | `{{ env "EDITOR" \| default "vim" }}` |
| `required`           | Mark value as required   | `{{ env "USER" \| required }}`        |

### System Information

| Function   | Description          | Example          |
| ---------- | -------------------- | ---------------- |
| `hostname` | Get hostname         | `{{ hostname }}` |
| `username` | Get current username | `{{ username }}` |
| `homedir`  | Get home directory   | `{{ homedir }}`  |
| `os`       | Get operating system | `{{ os }}`       |
| `arch`     | Get CPU architecture | `{{ arch }}`     |

### String Manipulation

| Function          | Description          | Example                                            |
| ----------------- | -------------------- | -------------------------------------------------- |
| `upper`           | Convert to uppercase | `{{ env "USER" \| upper }}`                        |
| `lower`           | Convert to lowercase | `{{ secret "EMAIL" \| lower }}`                    |
| `title`           | Title case           | `{{ env "PROJECT" \| title }}`                     |
| `trim`            | Remove whitespace    | `{{ env "VAR" \| trim }}`                          |
| `replace old new` | Replace string       | `{{ env "PATH" \| replace ":" " " }}`              |
| `contains substr` | Check substring      | `{{ if contains "darwin" os }}...{{ end }}`        |
| `hasPrefix`       | Check prefix         | `{{ if hasPrefix "/Users" homedir }}...{{ end }}`  |
| `hasSuffix`       | Check suffix         | `{{ if hasSuffix ".local" hostname }}...{{ end }}` |

### Path Operations

| Function    | Description          | Example                                   |
| ----------- | -------------------- | ----------------------------------------- |
| `abs`       | Absolute path        | `{{ abs "~/config" }}`                    |
| `base`      | Base name            | `{{ base "/path/to/file.txt" }}`          |
| `dir`       | Directory name       | `{{ dir "/path/to/file.txt" }}`           |
| `ext`       | File extension       | `{{ ext "file.txt" }}`                    |
| `clean`     | Clean path           | `{{ clean "//path//to///file" }}`         |
| `join_path` | Join path components | `{{ join_path homedir ".config" "app" }}` |

## Secret File Locations

dotgo looks for secrets in the following locations (in order):

1. `--secrets-file` flag (if specified)
2. `~/.config/dotgo/secrets/.env.local`
3. `~/.config/dotgo/secrets/.env`
4. System environment variables

## Examples

### SSH Configuration Template

```ssh-config
# ~/.ssh/config.tmpl
Host github.com
    User git
    Hostname github.com
    IdentityFile {{ secret "GITHUB_SSH_KEY" "~/.ssh/id_ed25519" }}

Host work-server
    User {{ secret "WORK_USERNAME" username }}
    Hostname {{ secret "WORK_SERVER" }}
    Port {{ secret "WORK_SSH_PORT" "22" }}
    {{ if hasSecret "WORK_PROXY" }}
    ProxyJump {{ secret "WORK_PROXY" }}
    {{ end }}
```

### AWS Configuration Template

```ini
# ~/.aws/config.tmpl
[default]
region = {{ secret "AWS_REGION" "us-east-1" }}
output = {{ secret "AWS_OUTPUT" "json" }}

{{ if hasSecret "AWS_PROFILE_WORK" }}
[profile work]
region = {{ secret "AWS_WORK_REGION" }}
role_arn = {{ secret "AWS_WORK_ROLE_ARN" }}
source_profile = default
{{ end }}
```

### NPM Configuration Template

```ini
# ~/.npmrc.tmpl
//registry.npmjs.org/:_authToken={{ secret "NPM_TOKEN" | required }}
{{ if hasSecret "CORPORATE_REGISTRY" }}
@corp:registry={{ secret "CORPORATE_REGISTRY" }}
//{{ secret "CORPORATE_REGISTRY" }}/:_authToken={{ secret "CORPORATE_TOKEN" }}
{{ end }}
```

## Package Configuration

When a template file is added, the package configuration is automatically updated:

```yaml
# ~/.dotfiles/packages/git/package.yaml
files:
  - source: gitconfig.tmpl
    target: ~/.gitconfig
    template: true
    template_vars:
      author: "{{ username }}"
    required_secrets:
      - GIT_USER_EMAIL
      - GIT_USER_NAME
```

## Best Practices

1. **Never Commit Secrets**: Keep `.env` files outside your dotfiles repository
2. **Use Required for Critical Values**: Mark essential secrets with `| required`
3. **Provide Sensible Defaults**: Use `default` for optional configuration
4. **Document Required Secrets**: List required environment variables in your README
5. **Use Git Includes**: For Git configuration, prefer `gitInclude` over templates for machine-specific settings
6. **Validate Early**: dotgo validates all required secrets before processing templates

## Security Considerations

- Secrets are stored in `~/.config/dotgo/secrets/` (outside the repository)
- Use file permissions (600) to protect secret files
- Never log or echo secret values in templates
- Consider using external secret managers for production environments
- Regularly rotate sensitive credentials

## Troubleshooting

### Missing Secrets Error

If you see an error about missing secrets:

```
Error: Required secrets missing: GIT_USER_EMAIL, GIT_SIGNING_KEY
Please create ~/.config/dotgo/secrets/.env with these variables
```

Create the specified file with the required variables.

### Template Syntax Errors

Template syntax errors show the line number and error:

```
Error: template parsing error at line 5: undefined function "secert"
Did you mean "secret"?
```

### Custom Secrets Location

Use the `--secrets-file` flag to specify a custom location:

```bash
dotgo install --secrets-file ~/.my-secrets/production.env git
```

## Migration Guide

### From Hardcoded Secrets

1. Copy your existing config: `cp ~/.gitconfig ~/.gitconfig.tmpl`
2. Replace sensitive values with template functions
3. Create `.env` file with the extracted values
4. Add template to dotgo: `dotgo add ~/.gitconfig.tmpl`
5. Test installation: `dotgo install git`

### From Git Include Pattern

If you're already using Git includes:

1. Keep using includes for machine-specific config
2. Use templates only for values that need transformation
3. Combine both approaches:

```gitconfig
# Template for processed values
[user]
    name = {{ secret "GIT_USER_NAME" }}

# Include for machine-specific settings
{{ gitInclude "~/.gitconfig.local" }}
```
