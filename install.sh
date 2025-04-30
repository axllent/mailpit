#!/bin/sh

# This script will install the latest version of Mailpit.

# Check dependencias is installed
for cmd in curl tar; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
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
TIMEOUT=90
# This is used to authenticate with the GitHub API. (Fix the public rate limiting issue)
# Try the GITHUB_TOKEN environment variable is set globally.
GITHUB_API_TOKEN="${GITHUB_TOKEN:-}"

# Update the default values if the user has set.
while [ $# -gt 0 ]; do
    case $1 in
    --install-path)
        shift
        case "$1" in
        */*)
            # Remove trailing slashes from the path.
            INSTALL_PATH="$(echo "$1" | sed 's#/\+$##')"
            [ -z "$INSTALL_PATH" ] && INSTALL_PATH="/"
            ;;
        esac
        ;;
    --auth | --auth-token | --github-token | --token)
        shift
        case "$1" in
        gh*)
            GITHUB_API_TOKEN="$1"
            ;;
        esac
        ;;
    *) ;;
    esac
    shift
done

# Description of the sort parameters for curl command.
# -s: Silent mode.
# -f: Fail silently on server errors.
# -L: Follow redirects.
# -m: Set maximum time allowed for the transfer.

if [ -n "$GITHUB_API_TOKEN" ] && [ "${#GITHUB_API_TOKEN}" -gt 36 ]; then
    CURL_OUTPUT="$(curl -sfL -m $TIMEOUT -H "Authorization: Bearer $GITHUB_API_TOKEN" https://api.github.com/repos/${GH_REPO}/releases/latest)"
    EXIT_CODE=$?
else
    CURL_OUTPUT="$(curl --sfL -m $TIMEOUT https://api.github.com/repos/${GH_REPO}/releases/latest)"
    EXIT_CODE=$?
fi

VERSION=""
if [ $EXIT_CODE -eq 0 ]; then
    # Extracts the latest version using jq, awk, or sed.
    if command -v jq >/dev/null 2>&1; then
        # Use jq -n because the output is not a valid JSON in sh.
        VERSION=$(jq -n "$CURL_OUTPUT" | jq -r '.tag_name')
    elif command -v awk >/dev/null 2>&1; then
        VERSION=$(echo "$CURL_OUTPUT" | awk -F: '$1 ~ /tag_name/ {gsub(/[^v0-9\.]+/, "", $2) ;print $2; exit}')
    elif command -v sed >/dev/null 2>&1; then
        VERSION=$(echo "$CURL_OUTPUT" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
    else
        EXIT_CODE=3
    fi
fi

# Validate the version.
case "$VERSION" in
v[0-9][0-9\.]*) ;;
*)
    echo "There was an error trying to check what is the latest version of Mailpit."
    echo "Please try again later."
    exit $EXIT_CODE
    ;;
esac

TEMP_DIR="$(mktemp -qd)"
EXIT_CODE=$?
# Ensure the temporary directory exists and is a directory.
if [ -z "$TEMP_DIR" ] || [ ! -d "$TEMP_DIR" ]; then
    echo "ERROR: Creating temporary directory."
    exit $EXIT_CODE
fi

GH_REPO_BIN="mailpit-${OS}-${OS_ARCH}.tar.gz"
if [ "$INSTALL_PATH" = "/" ]; then
    INSTALL_BIN_PATH="/mailpit"
else
    INSTALL_BIN_PATH="${INSTALL_PATH}/mailpit"
fi
cd "$TEMP_DIR" || EXIT_CODE=$?
if [ $EXIT_CODE -eq 0 ]; then
    # Download the latest release.
    #
    # Description of the sort parameters for curl command.
    # -s: Silent mode.
    # -f: Fail silently on server errors.
    # -L: Follow redirects.
    # -m: Set maximum time allowed for the transfer.
    # -o: Write output to a file instead of stdout.
    curl -sfL -m $TIMEOUT -o "${GH_REPO_BIN}" "https://github.com/${GH_REPO}/releases/download/${VERSION}/${GH_REPO_BIN}"
    EXIT_CODE=$?

    # The following conditions check each step of the installation.
    # If there is an error in any of the steps, an error message is printed.

    if [ $EXIT_CODE -eq 0 ]; then
        if ! [ -f "${GH_REPO_BIN}" ]; then
            EXIT_CODE=1
            echo "ERROR: Downloading latest release."
        fi
    fi

    if [ $EXIT_CODE -eq 0 ]; then
        tar zxf "$GH_REPO_BIN"
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
            echo "ERROR: Extracting \"${GH_REPO_BIN}\"."
        fi
    fi

    if [ $EXIT_CODE -eq 0 ] && [ ! -d "$INSTALL_PATH" ]; then
        mkdir -p "${INSTALL_PATH}"
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
            echo "ERROR: Creating \"${INSTALL_PATH}\" directory."
        fi
    fi

    if [ $EXIT_CODE -eq 0 ]; then
        cp mailpit "$INSTALL_BIN_PATH"
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
            echo "ERROR: Copying mailpit to \"${INSTALL_PATH}\" directory."
        fi
    fi

    if [ $EXIT_CODE -eq 0 ]; then
        chmod 755 "$INSTALL_BIN_PATH"
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
            echo "ERROR: Setting permissions for \"$INSTALL_BIN_PATH\" binary."
        fi
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

        chown "${OWNER}:${GROUP}" "$INSTALL_BIN_PATH"
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
            echo "ERROR: Setting ownership for \"$INSTALL_BIN_PATH\" binary."
        fi
    fi
else
    echo "ERROR: Changing to temporary directory."
    exit $EXIT_CODE
fi

# Cleanup the temporary directory.
rm -rf "$TEMP_DIR"
# Check the EXIT_CODE variable, and print the success or error message.
if [ $EXIT_CODE -ne 0 ]; then
    echo "There was an error installing Mailpit."
    echo "Please try again later."
    exit $EXIT_CODE
fi

echo "Installed successfully to \"$INSTALL_BIN_PATH\"."
exit 0
