#!/bin/sh
# Installs softether-tui from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/kh813/softether-tui/main/install.sh | sh
#   VERSION=v0.1.0 curl -fsSL .../install.sh | sh   # install a specific version
#
# Before piping this into `sh`, you can inspect it first:
#   curl -fsSL https://raw.githubusercontent.com/kh813/softether-tui/main/install.sh -o install.sh
#   less install.sh && sh install.sh
set -eu

REPO="${SOFTETHER_TUI_REPO:-kh813/softether-tui}"
VERSION="${VERSION:-latest}"
BIN_NAME="softether-tui"

log() { printf '%s\n' "$*" >&2; }
die() { log "error: $*"; exit 1; }

need() {
	command -v "$1" >/dev/null 2>&1 || die "'$1' is required but was not found in PATH"
}

need curl
need tar
need uname

detect_os() {
	case "$(uname -s)" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	*) die "unsupported OS: $(uname -s)" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
	x86_64 | amd64) echo amd64 ;;
	arm64 | aarch64) echo arm64 ;;
	*) die "unsupported architecture: $(uname -m)" ;;
	esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
ARCHIVE="${BIN_NAME}_${OS}_${ARCH}.tar.gz"

if [ "$VERSION" = "latest" ]; then
	BASE_URL="https://github.com/${REPO}/releases/latest/download"
else
	BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

log "Downloading ${ARCHIVE} (${VERSION}) from ${REPO}..."
curl -fsSL "${BASE_URL}/${ARCHIVE}" -o "${WORKDIR}/${ARCHIVE}" \
	|| die "failed to download ${BASE_URL}/${ARCHIVE} (check that ${REPO} has a release for ${VERSION})"

log "Downloading checksums.txt and verifying..."
if curl -fsSL "${BASE_URL}/checksums.txt" -o "${WORKDIR}/checksums.txt" 2>/dev/null; then
	EXPECTED="$(grep " ${ARCHIVE}\$" "${WORKDIR}/checksums.txt" | awk '{print $1}')"
	if [ -z "$EXPECTED" ]; then
		log "warning: ${ARCHIVE} not listed in checksums.txt, skipping verification"
	else
		if command -v sha256sum >/dev/null 2>&1; then
			ACTUAL="$(sha256sum "${WORKDIR}/${ARCHIVE}" | awk '{print $1}')"
		elif command -v shasum >/dev/null 2>&1; then
			ACTUAL="$(shasum -a 256 "${WORKDIR}/${ARCHIVE}" | awk '{print $1}')"
		else
			log "warning: no sha256sum/shasum found, skipping checksum verification"
			ACTUAL="$EXPECTED"
		fi
		[ "$EXPECTED" = "$ACTUAL" ] || die "checksum mismatch for ${ARCHIVE} (expected ${EXPECTED}, got ${ACTUAL})"
	fi
else
	log "warning: could not download checksums.txt, skipping verification"
fi

tar -xzf "${WORKDIR}/${ARCHIVE}" -C "${WORKDIR}" "${BIN_NAME}"

if [ "$(id -u)" = "0" ]; then
	INSTALL_DIR="/usr/local/bin"
else
	INSTALL_DIR="${HOME}/.local/bin"
fi
mkdir -p "$INSTALL_DIR"

install -m 0755 "${WORKDIR}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
log "Installed ${INSTALL_DIR}/${BIN_NAME}"

case ":$PATH:" in
*":${INSTALL_DIR}:"*) ;;
*)
	log ""
	log "${INSTALL_DIR} is not in your PATH. Add this to your shell profile:"
	log "  export PATH=\"${INSTALL_DIR}:\$PATH\""
	;;
esac

log ""
log "Run '${BIN_NAME} --version' to confirm the install."
