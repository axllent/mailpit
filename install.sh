#!/usr/bin/env bash

# This script will install the latest version of Mailpit.

# Check dependencias is installed
for cmd in curl tar; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Then $cmd command is required but not installed."
        echo "Please install $cmd and try again."
        exit 1
    fi
done

# Check if the OS is supported.
OS=
case "$(uname -s)" in
Linux) OS="linux" ;;
Darwin) OS="Darwin" ;;
*)
    echo "OS not supported."
    exit 2
    ;;
esac

# Detect the architecture of the OS.
OS_ARCH=
case "$(uname -m)" in
x86_64 | amd64)
    OS_ARCH="amd64"
    ;;
i?86 | x86)
    OS_ARCH="386"
    ;;
aarch64 | arm64)
    OS_ARCH="arm64"
    ;;
*)
    echo "OS architecture not supported."
    exit 2
    ;;
esac

GH_REPO="axllent/mailpit"
INSTALL_PATH="/usr/local/bin"
TIMEOUT=90 # --max-time in curl

# The arguments in extended format for the curl command.
CURL_ARGS=(
    "--silent"
    "--fail"
    "--location"
    "--max-time" "$TIMEOUT"
)
# This is used to authenticate with the GitHub API. (Fix the public rate limiting issue)
AUTH_TOKEN="${GITHUB_TOKEN:-}"

# Update the default values if the user has set.
while [[ $# -gt 0 ]]; do
    case $1 in
    --install-path)
        shift
        INSTALL_PATH="$1"
        ;;
    --auth | --auth-token | --github-token | --token)
        shift
        [[ "${1:-}" =~ ^- ]] || AUTH_TOKEN="$1"
        ;;
    *) ;;
    esac
    shift
done

# Set the header auth if the user has set a GitHub token.
if [[ -n "$AUTH_TOKEN" ]] && [[ "$AUTH_TOKEN" =~ ^gh[pousr]_[A-Za-z0-9_]{36,251}$ ]]; then
    CURL_ARGS+=("--header" "Authorization: 'Bearer $AUTH_TOKEN'")
fi

CURL_OUTPUT=$(curl "${CURL_ARGS[@]}" "https://api.github.com/repos/${GH_REPO}/releases/latest")
EXIT_CODE=$?
if [[ $EXIT_CODE -eq 0 ]]; then
    # Extracts the latest version using jq, awk, or sed.
    if command -v jq &>/dev/null; then
        VERSION=$(echo "$CURL_OUTPUT" | jq -r '.tag_name')
    elif command -v awk &>/dev/null; then
        VERSION=$(echo "$CURL_OUTPUT" | awk -F: '$1 ~ /tag_name/ {gsub(/[^v0-9\.]+/, "", $2) ;print $2; exit}')
    else
        VERSION=$(echo "$CURL_OUTPUT" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
    fi
fi

# Validate the version.
if ! [[ "$VERSION" =~ ^v[0-9]{1,}[0-9\.]+$ ]]; then
    echo "There was an error trying to check what is the latest version of Mailpit."
    echo "Please try again later."
    exit $EXIT_CODE
fi

TEMP_DIR="$(mktemp -qd)"
EXIT_CODE=$?
# Ensure the temporary directory exists and is a directory.
if [ -z "$TEMP_DIR" ] || [ ! -d "$TEMP_DIR" ]; then
    echo "ERROR: Creating temporary directory."
    exit $EXIT_CODE
fi

GH_REPO_BIN="mailpit-${OS}-${OS_ARCH}.tar.gz"
CURL_ARGS+=("--output" "${GH_REPO_BIN}")

if ! cd "$TEMP_DIR"; then
    EXIT_CODE=$?
    echo "ERROR: Changing to temporary directory."
else
    # Download the latest release.
    curl "${CURL_ARGS[@]}" "https://github.com/${GH_REPO}/releases/download/${VERSION}/${GH_REPO_BIN}"
    EXIT_CODE=$?

    # The following conditions check each step of the installation.
    # If there is an error in any of the steps, an error message is printed.

    if ! [[ -f "${GH_REPO_BIN}" ]]; then
        echo "ERROR: Downloading latest release."
    elif ! tar zxf "$GH_REPO_BIN"; then
        EXIT_CODE=$?
        echo "ERROR: Extracting \"${GH_REPO_BIN}\"."
    elif ! mkdir -p "${INSTALL_PATH}"; then
        EXIT_CODE=$?
        echo "ERROR: Creating \"${INSTALL_PATH}\" directory."
    elif ! cp -f mailpit "${INSTALL_PATH}"; then
        EXIT_CODE=$?
        echo "ERROR: Copying mailpit to \"${INSTALL_PATH}\" directory."
    elif ! chmod 755 "${INSTALL_PATH}/mailpit"; then
        EXIT_CODE=$?
        echo "ERROR: Setting permissions for \"${INSTALL_PATH}/mailpit\" binary."
    fi

    # Set the owner and group to root:root if the script is run as root.
    if [ $EXIT_CODE -eq 0 ] && [ "$(id -u)" -eq "0" ]; then
        OWNER="root"
        GROUP="root"
        # Set the OWNER, GROUP variable when the OS not use the default root:root.
        case "$OS" in
        darwin) GROUP="wheel" ;;
        *) ;;
        esac

        if ! chown "${OWNER}:${GROUP}" "${INSTALL_PATH}/mailpit"; then
            EXIT_CODE=$?
            echo "ERROR: Setting ownership for \"${INSTALL_PATH}/mailpit\" binary."
        fi
    fi
fi

# Cleanup the temporary directory.
rm -rf "$TEMP_DIR"
# Check the EXIT_CODE variable, and print the success or error message.
if [[ $EXIT_CODE -eq 0 ]]; then
    echo "Installed successfully to \"${INSTALL_PATH}/mailpit\""
else
    echo "There was an error installing Mailpit."
    echo "Please try again later."
fi
exit $EXIT_CODE
