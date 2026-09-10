#!/bin/sh
set -e

REPO="octopyid/rune"

# Detect OS
case "$(uname -s)" in
    Linux*)     OS="linux" ;;
    Darwin*)    OS="darwin" ;;
    *)          echo "Error: Unsupported operating system $(uname -s)" >&2; exit 1 ;;
esac

# Detect Architecture
case "$(uname -m)" in
    x86_64|amd64)   ARCH="amd64" ;;
    arm64|aarch64)  ARCH="arm64" ;;
    *)              echo "Error: Unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac

BINARY="rune-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"

# Determine installation directory
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    if [ -n "$SUDO_USER" ] || [ "$(id -u)" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="${HOME}/.local/bin"
        mkdir -p "$INSTALL_DIR"
    fi
fi

TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

echo "Downloading Rune (${OS}/${ARCH})..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMP_FILE"
elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMP_FILE" "$URL"
else
    echo "Error: Neither curl nor wget was found." >&2
    exit 1
fi

chmod +x "$TMP_FILE"

echo "Installing rune to ${INSTALL_DIR}/rune..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "${INSTALL_DIR}/rune"
else
    sudo mv "$TMP_FILE" "${INSTALL_DIR}/rune"
fi

echo "Rune installed successfully!"
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "Note: Make sure ${INSTALL_DIR} is in your PATH."
fi
