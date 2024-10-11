# Secret Workflow Companion (Golang)

[![Build and Test ghm](https://github.com/Cdaprod/Secret-Workflow-Companion-Go/actions/workflows/ci.yml/badge.svg)](https://github.com/Cdaprod/Secret-Workflow-Companion-Go/actions/workflows/ci.yml)

---

[Alternate Python Version](https://github.com/cdaprod/secret-workflow-companion-py)

---

A command-line interface tool written in Go for managing GitHub secrets and workflows, ensuring secure and efficient operations across repositories.

## Features

1. Interactive CLI Interface: User-friendly menu for selecting actions.
2. GitHub Secrets Management: Add and list secrets for GitHub repositories.
3. GitHub Actions Workflow Management: Add and list workflow files.
4. Configuration Storage: Store and retrieve arbitrary configurations.
5. Colorful Output: Enhanced user experience with color-coded prompts and messages.
6. Persistent Storage: Local storage of secrets and configurations for future reference.
7. GitHub CLI Integration: Utilizes GitHub CLI for seamless interaction with GitHub.
8. Autocomplete: Bash/Zsh autocomplete for easier command input.

## Prerequisites

- Go 1.16 or later
- GitHub CLI (gh)
- Git

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/Cdaprod/secret-workflow-companion.git
   cd secret-workflow-companion
   ```

2. Build and install the tool:
   ```
   make install
   ```

3. Reload your shell or run:
   ```
   source ~/.bash_profile
   ```
   (or `source ~/.zshrc` if you're using Zsh)

The installation process will:
- Build the binary and place it in `~/bin/`
- Install the autocomplete script
- Add the necessary source line to your `.bash_profile` and `.zshrc`

## Usage

Run the tool by typing:

```
ghmanager
```

The tool supports autocomplete for its commands. Type `ghmanager` followed by a space and press Tab to see available options.

Follow the interactive prompts to:

1. Add a Secret
2. Add a GitHub Actions Workflow
3. Store a Configuration
4. Exit

## Examples

### Adding a Secret

```
ghmanager secret add
```

### Adding a Workflow

```
ghmanager workflow add
```

### Storing a Configuration

```
ghmanager config store
```

## Development

- Build the project:
  ```
  make build
  ```

- Run tests:
  ```
  make test
  ```

- Clean build artifacts:
  ```
  make clean
  ```

## Uninstallation

To uninstall the tool, run:

```
make uninstall
```

This will remove the binary, autocomplete script, and the source lines from your `.bash_profile` and `.zshrc`.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.