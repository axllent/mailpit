#!/usr/bin/env bash

GH_REPO="axllent/mailpit"
TIMEOUT=90
INSTALL_PATH="${INSTALL_PATH:-/usr/local/bin}"

set -e

VERSION=$(curl --silent --fail --location --max-time "${TIMEOUT}" "https://api.github.com/repos/${GH_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ $? -ne 0 ]; then
    echo -ne "\nThere was an error trying to check what is the latest version of Mailpit.\nPlease try again later.\n"
    exit 1
fi

# detect the platform
OS="$(uname)"
case $OS in
Linux)
    OS='linux'
    ;;
FreeBSD)
    OS='freebsd'
    echo 'OS not supported'
    exit 2
    ;;
NetBSD)
    OS='netbsd'
    echo 'OS not supported'
    exit 2
    ;;
OpenBSD)
    OS='openbsd'
    echo 'OS not supported'
    exit 2
    ;;
Darwin)
    OS='darwin'
    ;;
SunOS)
    OS='solaris'
    echo 'OS not supported'
    exit 2
    ;;
*)
    echo 'OS not supported'
    exit 2
    ;;
esac

# detect the arch
OS_type="$(uname -m)"
case "$OS_type" in
x86_64 | amd64)
    OS_type='amd64'
    ;;
i?86 | x86)
    OS_type='386'
    ;;
aarch64 | arm64)
    OS_type='arm64'
    ;;
*)
    echo 'OS type not supported'
    exit 2
    ;;
esac

GH_REPO_BIN="mailpit-${OS}-${OS_type}.tar.gz"

#create tmp directory and move to it with macOS compatibility fallback
tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'mailpit-install.XXXXXXXXXX')
cd "$tmp_dir"

echo "Downloading Mailpit $VERSION"
LINK="https://github.com/${GH_REPO}/releases/download/${VERSION}/${GH_REPO_BIN}"

curl --silent --fail --location --max-time "${TIMEOUT}" "${LINK}" -o "${GH_REPO_BIN}" || {
    echo "Error downloading latest release"
    exit 2
}

tar zxf "$GH_REPO_BIN" || {
    echo "Error extracting ${GH_REPO_BIN}"
    exit 2
}

mkdir -p "${INSTALL_PATH}" || exit 2
cp mailpit "${INSTALL_PATH}" || exit 2
chmod 755 "${INSTALL_PATH}/mailpit" || exit 2
case "$OS" in
'linux')
    if [ "$(id -u)" -eq "0" ]; then
        chown root:root "${INSTALL_PATH}/mailpit" || exit 2
    fi
    ;;
'freebsd' | 'openbsd' | 'netbsd' | 'darwin')
    if [ "$(id -u)" -eq "0" ]; then
        chown root:wheel "${INSTALL_PATH}/mailpit" || exit 2
    fi
    ;;
*)
    echo 'OS not supported'
    exit 2
    ;;
esac

rm -rf "$tmp_dir"
echo "Installed successfully to ${INSTALL_PATH}/mailpit"
