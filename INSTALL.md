# OpenAPI CLI Installation Guide

This guide provides detailed installation instructions for the OpenAPI CLI tool across different platforms and methods.

## Table of Contents

- [Recommended Methods](#recommended-methods)
- [Script-Based Installation](#script-based-installation)
- [Manual Installation](#manual-installation)
- [Custom Installation Options](#custom-installation-options)
- [Upgrading](#upgrading)
- [Troubleshooting](#troubleshooting)

## Recommended Methods

### Homebrew (macOS/Linux)

The easiest way to install on macOS or Linux with Homebrew:

```bash
brew install openapi
```

To upgrade to the latest version:

```bash
brew upgrade openapi
```

### Go Install

If you have Go installed, you can install directly:

```bash
go install github.com/speakeasy-api/openapi/cmd/openapi@latest
```

This installs the binary to your `$GOPATH/bin` directory (typically `~/go/bin`).

## Script-Based Installation

For quick installation without package managers, use our installation scripts.

### Linux/macOS

```bash
curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

Or with wget:

```bash
wget -qO- https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 | iex
```

Or the long form:

```powershell
Invoke-WebRequest -Uri https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 -UseBasicParsing | Invoke-Expression
```

### Windows (Git Bash/MSYS2/Cygwin)

```bash
curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

## Manual Installation

1. Visit the [latest release page](https://github.com/speakeasy-api/openapi/releases/latest)
2. Download the appropriate archive for your platform:
   - **Linux (x86_64):** `openapi_Linux_x86_64.tar.gz`
   - **Linux (ARM64):** `openapi_Linux_arm64.tar.gz`
   - **macOS (x86_64):** `openapi_Darwin_x86_64.tar.gz`
   - **macOS (ARM64):** `openapi_Darwin_arm64.tar.gz`
   - **Windows (x86_64):** `openapi_Windows_x86_64.zip`
   - **Windows (ARM64):** `openapi_Windows_arm64.zip`
3. Extract the archive
4. Move the `openapi` binary to a directory in your PATH

### Example (Linux/macOS)

```bash
# Download (replace with actual version URL)
curl -LO https://github.com/speakeasy-api/openapi/releases/latest/download/openapi_Linux_x86_64.tar.gz

# Extract
tar -xzf openapi_Linux_x86_64.tar.gz

# Move to PATH
sudo mv openapi /usr/local/bin/

# Verify installation
openapi --help
```

### Example (Windows - PowerShell)

```powershell
# Download (replace with actual version URL)
Invoke-WebRequest -Uri "https://github.com/speakeasy-api/openapi/releases/latest/download/openapi_Windows_x86_64.zip" -OutFile "openapi.zip"

# Extract
Expand-Archive -Path openapi.zip -DestinationPath .

# Move to a directory in your PATH (example)
Move-Item openapi.exe $env:LOCALAPPDATA\Programs\

# Add to PATH if needed
$path = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", "$path;$env:LOCALAPPDATA\Programs", "User")
```

## Custom Installation Options

Both installation scripts support custom configuration via environment variables.

### Linux/macOS/Git Bash

**Environment Variables:**
- `OPENAPI_INSTALL_DIR` - Installation directory (default: `/usr/local/bin`)
- `OPENAPI_VERSION` - Specific version to install (default: `latest`)

**Examples:**

```bash
# Install to a custom directory
OPENAPI_INSTALL_DIR=$HOME/.local/bin curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash

# Install a specific version
OPENAPI_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash

# Combine both options
OPENAPI_INSTALL_DIR=$HOME/bin OPENAPI_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

### Windows (PowerShell)

**Environment Variables:**
- `$env:OPENAPI_INSTALL_DIR` - Installation directory (default: `$env:LOCALAPPDATA\Programs\OpenAPI`)
- `$env:OPENAPI_VERSION` - Specific version to install (default: `latest`)

**Examples:**

```powershell
# Install to a custom directory
$env:OPENAPI_INSTALL_DIR = "$env:USERPROFILE\.openapi"
iwr -useb https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 | iex

# Install a specific version
$env:OPENAPI_VERSION = "v1.0.0"
iwr -useb https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 | iex

# Combine both options
$env:OPENAPI_INSTALL_DIR = "$env:USERPROFILE\.openapi"
$env:OPENAPI_VERSION = "v1.0.0"
iwr -useb https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 | iex
```

## Upgrading

### Homebrew

```bash
brew upgrade openapi
```

### Go Install

Simply reinstall to get the latest version:

```bash
go install github.com/speakeasy-api/openapi/cmd/openapi@latest
```

### Script-Based Installation

The installation scripts always install the latest version by default, so simply run them again to upgrade.

**Linux/macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

**Windows (PowerShell):**

```powershell
iwr -useb https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 | iex
```

**Windows (Git Bash):**

```bash
curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

The script will automatically replace the existing binary with the latest version.

### Upgrade to Specific Version

To upgrade (or downgrade) to a specific version, use the version environment variable:

**Linux/macOS/Git Bash:**

```bash
OPENAPI_VERSION=v1.2.0 curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

**Windows (PowerShell):**

```powershell
$env:OPENAPI_VERSION = "v1.2.0"
iwr -useb https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 | iex
```

### Manual Upgrade

If you installed manually, follow the same [manual installation steps](#manual-installation) to download and replace the binary with the newer version.

### Checking Your Current Version

To see which version you currently have installed:

```bash
openapi --version
```

## Troubleshooting

### Command Not Found After Installation

If you get "command not found" after installation, the installation directory may not be in your PATH.

**Linux/macOS:**

Add the installation directory to your PATH by adding this line to your `~/.bashrc`, `~/.zshrc`, or equivalent:

```bash
export PATH="$PATH:/usr/local/bin"
```

Or for custom installation directories:

```bash
export PATH="$PATH:$HOME/.local/bin"
```

Then reload your shell configuration:

```bash
source ~/.bashrc  # or ~/.zshrc
```

**Windows:**

The PowerShell script automatically adds the installation directory to your PATH. If it's still not working:

1. Restart your terminal
2. Or run `refreshenv` (if you have Chocolatey installed)
3. Or manually add the directory to your PATH in System Environment Variables

### Permission Denied (Linux/macOS)

If you get permission errors when installing to `/usr/local/bin`:

**Option 1:** Run with sudo (not recommended for the curl pipe method)

```bash
# Download script first, review it, then run with sudo
curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh -o install.sh
chmod +x install.sh
sudo ./install.sh
```

**Option 2:** Install to a user-writable directory

```bash
OPENAPI_INSTALL_DIR=$HOME/.local/bin curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash
```

Then ensure `$HOME/.local/bin` is in your PATH.

### Windows Git Bash Issues

If you're using Git Bash on Windows and encounter issues:

1. Ensure you have `unzip` installed (may need to install via Git for Windows options)
2. Or use the PowerShell script instead for a smoother experience

### Version-Specific Installation

To install a specific version, use the version environment variable:

```bash
# Linux/macOS
OPENAPI_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.sh | bash

# Windows (PowerShell)
$env:OPENAPI_VERSION = "v1.0.0"
iwr -useb https://raw.githubusercontent.com/speakeasy-api/openapi/main/scripts/install.ps1 | iex
```

### Verifying Installation

After installation, verify it worked:

```bash
openapi --help
```

You should see the help output for the OpenAPI CLI tool.

### Uninstalling

**Homebrew:**
```bash
brew uninstall openapi
```

**Go Install:**
```bash
rm $(which openapi)
```

**Script Installation (Linux/macOS):**
```bash
sudo rm /usr/local/bin/openapi
# Or if you used a custom directory:
rm $HOME/.local/bin/openapi
```

**Script Installation (Windows - PowerShell):**
```powershell
Remove-Item "$env:LOCALAPPDATA\Programs\OpenAPI\openapi.exe"
```

## Getting Help

If you encounter issues not covered here:

1. Check the [GitHub Issues](https://github.com/speakeasy-api/openapi/issues)
2. Open a new issue with details about your platform and the error
3. Join our [Slack community](https://go.speakeasy.com/slack) for support