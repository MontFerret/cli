#!/bin/bash
# Copyright MontFerret Team 2023
# Licensed under the MIT license.

set -e

# Declare constants
readonly projectName="MontFerret"
readonly appName="cli"
readonly binName="ferret"
readonly fullAppName="Ferret CLI"
readonly baseUrl="https://github.com/${projectName}/${appName}/releases/download"

# Declare default values
readonly defaultLocation="${HOME}/.ferret"
readonly defaultVersion="latest"

# Global flags
DRY_RUN=false
VERBOSE=false
HELP=false
AUTO_CREATE_DIR=false
UNINSTALL=false

# Print a message to stdout
report() {
  command printf "%s\n" "$*" 2>/dev/null
}

# Print verbose message if verbose mode is enabled
verbose() {
  if [ "$VERBOSE" = true ]; then
    report "[VERBOSE] $*"
  fi
}

# Print usage information
show_help() {
  cat << EOF
${fullAppName} Installation Script

USAGE:
    install.sh [OPTIONS]

DESCRIPTION:
    Downloads and installs the latest version of ${fullAppName} to a specified location.
    By default, installs to ${defaultLocation} and adds it to PATH.

OPTIONS:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose output
    -d, --dry-run           Show what would be done without actually installing
    -u, --uninstall         Uninstall ${fullAppName}
    -c, --create-dir        Automatically create install directory if it doesn't exist
    -l, --location PATH     Install location (default: ${defaultLocation})
    -V, --version VERSION   Version to install (default: ${defaultVersion})

ENVIRONMENT VARIABLES:
    LOCATION                Install location (overridden by --location)
    VERSION                 Version to install (overridden by --version)

EXAMPLES:
    # Install to default location
    ./install.sh

    # Install to custom location
    ./install.sh --location /usr/local/bin

    # Install specific version
    ./install.sh --version v1.2.3

    # Dry run to see what would happen
    ./install.sh --dry-run

    # Auto-create directory if needed
    ./install.sh --location /opt/ferret --create-dir

    # Uninstall
    ./install.sh --uninstall

For more information, visit: https://github.com/${projectName}/${appName}
EOF
}

# Parse command line arguments
parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
      -h|--help)
        HELP=true
        shift
        ;;
      -v|--verbose)
        VERBOSE=true
        shift
        ;;
      -d|--dry-run)
        DRY_RUN=true
        shift
        ;;
      -u|--uninstall)
        UNINSTALL=true
        shift
        ;;
      -c|--create-dir)
        AUTO_CREATE_DIR=true
        shift
        ;;
      -l|--location)
        LOCATION="$2"
        shift 2
        ;;
      -V|--version)
        VERSION="$2"
        shift 2
        ;;
      *)
        report "Unknown option: $1"
        report "Use --help to see available options."
        exit 1
        ;;
    esac
  done
}

# Check if a command is available
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Check if a path exists
check_path() {
  if [ -z "${1-}" ] || [ ! -f "${1}" ]; then
    return 1
  fi

  report "${1}"
}

# Validate user input
validate_input() {
  local location="$1"
  local version="$2"

  verbose "Validating location: $location"
  verbose "Validating version: $version"

  if [ -z "$location" ]; then
    report "Error: Invalid location: $location"
    report "Please provide a valid installation directory."
    exit 1
  fi

  # Check if location exists
  if [ ! -d "$location" ]; then
    if [ "$AUTO_CREATE_DIR" = true ]; then
      verbose "Auto-creating directory: $location"
      if [ "$DRY_RUN" = false ]; then
        mkdir -p "$location" || {
          report "Error: Failed to create directory: $location"
          report "Please check permissions or create the directory manually."
          exit 1
        }
      else
        report "Would create directory: $location"
      fi
    else
      report "Error: Directory does not exist: $location"
      report "Use --create-dir to automatically create it, or create it manually:"
      report "  mkdir -p \"$location\""
      exit 1
    fi
  fi

  # Check if location is writable (only if not dry run)
  if [ "$DRY_RUN" = false ] && [ ! -w "$location" ]; then
    report "Error: Directory is not writable: $location"
    report "Please check permissions or choose a different location."
    exit 1
  fi

  if [ "$version" != "latest" ]; then
    # Remove leading 'v' if present
    version="${version#v}"

    # Check if version is valid using grep
    if ! echo "$version" | grep -qE "^[0-9]+\.[0-9]+\.[0-9]+$"; then
      report "Error: Invalid version format: $version"
      report "Version should be in format 'X.Y.Z' or 'vX.Y.Z' (e.g., '1.2.3' or 'v1.2.3')"
      exit 1
    fi
  fi
}

# Detect the profile file
detect_profile() {
  local profile=""
  local detected_profile=""

  if [ "${PROFILE-}" = '/dev/null' ]; then
    # the user has specifically requested NOT to have us touch their profile
    return
  fi

  if [ -n "${PROFILE}" ] && [ -f "${PROFILE}" ]; then
    report "${PROFILE}"
    return
  fi

  if command_exists bash; then
    if [ -f "$HOME/.bashrc" ]; then
      detected_profile="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
      detected_profile="$HOME/.bash_profile"
    fi
  elif command_exists zsh; then
    if [ -f "$HOME/.zshrc" ]; then
      detected_profile="$HOME/.zshrc"
    fi
  fi

  if [ -z "$detected_profile" ]; then
    for profile_name in ".zshrc" ".bashrc" ".bash_profile" ".profile"; do
      if detected_profile="$(check_path "${HOME}/${profile_name}")"; then
        break
      fi
    done
  fi

  if [ -n "$detected_profile" ]; then
    report "$detected_profile"
  fi
}

# Update the profile file
update_profile() {
  local location="$1"
  local profile="$(detect_profile)"

  if [ -z "$profile" ]; then
    report "No profile found. Skipping PATH update."
    return
  fi

  report "Checking if $location is already in PATH"

  if echo ":$PATH:" | grep -q ":$location:"; then
    report "$location is already in PATH"
    return
  fi

  report "Updating profile $profile"

  if [ -z "$profile" ]; then
    report "Profile not found. Tried ${DETECTED_PROFILE-} (as defined in \$PROFILE), ~/.bashrc, ~/.bash_profile, ~/.zshrc, and ~/.profile."
    report "Append the following lines to the correct file yourself:"
    report
    report "export PATH=\$PATH:${location}"
    report
  else
    if ! grep -q "${location}" "$profile"; then
      report "export PATH=\$PATH:${location}" >>"$profile"
    fi
  fi
}

# Get the platform-specific filename suffix
get_platform_suffix() {
  local platform_name="$(uname)"
  local arch_name="$(uname -m)"
  local platform=""
  local arch=""

  verbose "Detected platform: $platform_name"
  verbose "Detected architecture: $arch_name"

  case "$platform_name" in
  "Darwin")
    platform="_darwin"
    ;;
  "Linux")
    platform="_linux"
    ;;
  "Windows" | "MINGW"* | "MSYS"* | "CYGWIN"*)
    platform="_windows"
    ;;
  *)
    report "Error: Unsupported platform: $platform_name"
    report "Supported platforms: Linux, macOS (Darwin), Windows"
    exit 1
    ;;
  esac

  case "$arch_name" in
  "x86_64" | "amd64")
    arch="_x86_64"
    ;;
  "aarch64" | "arm64")
    arch="_arm64"
    ;;
  "i386" | "i686")
    arch="_i386"
    ;;
  *)
    report "Error: Unsupported architecture: $arch_name"
    report "Supported architectures: x86_64/amd64, aarch64/arm64, i386/i686"
    exit 1
    ;;
  esac

  echo "${platform}${arch}"
}

get_version_tag() {
  local version="$1"

  if [ "$version" = "latest" ]; then
    local url="https://api.github.com/repos/${projectName}/${appName}/releases/latest"
    
    verbose "Fetching latest version from: $url"
    
    # Try to get latest version from GitHub API with better error handling
    local tag_name=""
    if command_exists curl; then
      tag_name=$(curl -sSL "${url}" 2>/dev/null | grep "tag_name" | cut -d '"' -f 4)
    fi
    
    # If API call failed or returned empty, use a fallback
    if [ -z "$tag_name" ]; then
      report "Warning: Could not fetch latest version from GitHub API"
      report "This might be due to network connectivity or rate limiting"
      report "Please specify a version manually with --version, or try again later"
      exit 1
    fi
    
    verbose "Latest version found: $tag_name"
    echo "$tag_name"
  else
    # Check if the version starts with a 'v'
    if [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "$version"
    else
      echo "v$version"
    fi
  fi
}

# Install the package
install() {
  local location="${LOCATION:-$defaultLocation}"
  local version_input="${VERSION:-$defaultVersion}"
  local version
  local tmp_dir

  verbose "Starting installation process"
  verbose "Target location: $location"
  verbose "Requested version: $version_input"

  # Get the actual version tag
  version=$(get_version_tag "$version_input")
  verbose "Resolved version: $version"

  validate_input "$location" "$version_input"

  if [ "$DRY_RUN" = true ]; then
    report "DRY RUN - Would perform the following actions:"
    report "  1. Create temporary directory"
    report "  2. Download ${projectName} ${appName} ${version}"
    report "  3. Extract and install to: ${location}"
    report "  4. Make executable: ${location}/${binName}"
    report "  5. Update PATH in shell profile"
    report "  6. Clean up temporary files"
    report ""
    report "To proceed with installation, run without --dry-run"
    return 0
  fi

  tmp_dir="$(mktemp -d -t "${projectName}.${appName}.XXXXXXX")"
  verbose "Created temporary directory: $tmp_dir"

  # Ensure cleanup on exit
  trap 'rm -rf "$tmp_dir"' EXIT

  report "Installing ${projectName} ${appName} ${version}..."

  # Download the archive to a temporary location
  local suffix="$(get_platform_suffix)"
  local file_name="${appName}${suffix}"
  local download_dir="${tmp_dir}/${file_name}@${version}"

  verbose "Platform suffix: $suffix"
  verbose "File name: $file_name"

  mkdir -p "${download_dir}"

  local download_file="${download_dir}/${file_name}.tar.gz"
  local url="${baseUrl}/${version}/${file_name}.tar.gz"

  report "Downloading package from $url"
  verbose "Saving to: $download_file"

  # Download with better error handling
  if ! curl -sSL "${url}" | tar xz --directory "${download_dir}"; then
    report "Error: Failed to download or extract package from $url"
    report "Please check:"
    report "  1. Your internet connection"
    report "  2. That version ${version} exists"
    report "  3. That your system architecture is supported"
    exit 1
  fi

  local downloaded_file="${download_dir}/${binName}"
  
  if [ ! -f "${downloaded_file}" ]; then
    report "Error: Downloaded file not found: ${downloaded_file}"
    report "The download may have failed or the archive structure is unexpected"
    exit 1
  fi

  verbose "Successfully downloaded: ${downloaded_file}"

  report "Installing to ${location}/${binName}"

  if ! cp "${downloaded_file}" "${location}"; then
    report "Error: Failed to copy binary to ${location}"
    report "Please check write permissions for the target directory"
    exit 1
  fi

  local executable="${location}/${binName}"

  if ! chmod +x "${executable}"; then
    report "Error: Failed to make binary executable: ${executable}"
    exit 1
  fi

  verbose "Made executable: ${executable}"

  update_profile "${location}"

  report "✅ ${fullAppName} ${version} successfully installed to ${location}"
  report ""

  # Test the installation
  if "${executable}" version >/dev/null 2>&1; then
    report "Installation verified successfully:"
    "${executable}" version
  else
    report "Warning: Installation completed but binary test failed"
    report "You may need to restart your shell or run: source ~/.bashrc"
  fi

  report ""
  report "You can now use '${binName}' command (restart your shell if needed)"
}

# Uninstall the package
uninstall() {
  local location="${LOCATION:-$defaultLocation}"
  local executable="${location}/${binName}"
  
  verbose "Starting uninstall process"
  verbose "Target location: $location"
  verbose "Executable path: $executable"

  if [ "$DRY_RUN" = true ]; then
    report "DRY RUN - Would perform the following actions:"
    if [ -f "$executable" ]; then
      report "  1. Remove executable: $executable"
    else
      report "  1. Executable not found at: $executable"
    fi
    report "  2. Note: PATH entries in shell profiles will remain (manual cleanup required)"
    report ""
    report "To proceed with uninstall, run without --dry-run"
    return 0
  fi

  if [ ! -f "$executable" ]; then
    report "❌ ${fullAppName} is not installed at: $executable"
    report "If installed elsewhere, use --location to specify the correct path"
    exit 1
  fi

  report "Uninstalling ${fullAppName} from $location..."

  if rm "$executable"; then
    report "✅ ${fullAppName} successfully uninstalled from $location"
    report ""
    report "Note: PATH entries in your shell profile were not modified."
    report "You may want to manually remove this line from your profile:"
    report "export PATH=\$PATH:$location"
  else
    report "❌ Failed to remove: $executable"
    report "Please check permissions or remove manually"
    exit 1
  fi
}

# Call the main function
main() {
  # Parse command line arguments
  parse_args "$@"

  # Show help if requested
  if [ "$HELP" = true ]; then
    show_help
    exit 0
  fi

  # Set verbose mode early if needed
  if [ "$VERBOSE" = true ]; then
    verbose "Verbose mode enabled"
  fi

  # Handle uninstall
  if [ "$UNINSTALL" = true ]; then
    uninstall
    exit 0
  fi

  # Run installation
  install
}

# Only run main if script is executed directly (not sourced)
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  main "$@"
fi
