#!/bin/sh
# forgesync installer
#
# Usage:
#   curl -fsSL https://github.com/nox456/forgesync/releases/latest/download/install.sh | sh
#
# Optional environment variables:
#   FORGESYNC_VERSION       Release tag to install, e.g. v0.1.0 (default: latest)
#   FORGESYNC_INSTALL_DIR   Install directory (default: /usr/local/bin)
#
# Everything runs inside main(), which is only invoked on the very last line.
# If the download is truncated mid-pipe, the shell reaches EOF before that call
# and executes nothing.

# --- small helpers (definitions only — no side effects) --------------------

err()  { printf 'error: %s\n' "$1" >&2; exit 1; }
info() { printf '%s\n' "$1" >&2; }                 # logs go to stderr, never stdout
have() { command -v "$1" >/dev/null 2>&1; }

main() {
  set -eu

  REPO="nox456/forgesync"
  BINARY="forgesync"
  INSTALL_DIR="${FORGESYNC_INSTALL_DIR:-/usr/local/bin}"

  # --- pick a downloader ---------------------------------------------------

  if have curl; then
    download() { curl -fsSL "$1" -o "$2"; }
    fetch()    { curl -fsSL "$1"; }
  elif have wget; then
    download() { wget -qO "$2" "$1"; }
    fetch()    { wget -qO - "$1"; }
  else
    err "need either curl or wget installed"
  fi

  # --- detect platform -----------------------------------------------------

  os="$(uname -s)"
  case "$os" in
    Linux)  os="linux" ;;
    Darwin) os="darwin" ;;
    *) err "unsupported OS '$os' (this installer supports Linux and macOS)" ;;
  esac

  arch="$(uname -m)"
  case "$arch" in
    x86_64 | amd64)  arch="amd64" ;;
    aarch64 | arm64) arch="arm64" ;;
    *) err "unsupported architecture '$arch'" ;;
  esac

  # --- resolve the version -------------------------------------------------

  tag="${FORGESYNC_VERSION:-}"
  if [ -z "$tag" ]; then
    info "Resolving latest release..."
    tag="$(fetch "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name":' \
      | head -n1 \
      | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
    [ -n "$tag" ] || err "could not determine the latest release tag"
  fi

  # GoReleaser drops the leading 'v' from the version inside the filename,
  # but the release URL path keeps the full tag.
  version="${tag#v}"
  archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
  base="https://github.com/${REPO}/releases/download/${tag}"

  info "Installing ${BINARY} ${tag} (${os}/${arch})"

  # --- download into a temp dir --------------------------------------------

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  download "${base}/${archive}" "${tmp}/${archive}" \
    || err "failed to download ${archive} (does this release ship ${os}/${arch}?)"

  # --- verify the checksum -------------------------------------------------

  if download "${base}/checksums.txt" "${tmp}/checksums.txt" 2>/dev/null; then
    if have sha256sum; then
      sumcmd="sha256sum"
    elif have shasum; then
      sumcmd="shasum -a 256"
    else
      sumcmd=""
    fi

    if [ -n "$sumcmd" ]; then
      expected="$(awk -v f="$archive" '$2 == f { print $1 }' "${tmp}/checksums.txt")"
      actual="$(cd "$tmp" && $sumcmd "$archive" | awk '{ print $1 }')"
      [ -n "$expected" ] || err "no checksum entry for ${archive}"
      [ "$expected" = "$actual" ] || err "checksum mismatch for ${archive}"
      info "Checksum verified."
    else
      info "warning: no sha256 tool found, skipping checksum verification"
    fi
  else
    info "warning: checksums.txt not found, skipping checksum verification"
  fi

  # --- extract -------------------------------------------------------------

  tar -xzf "${tmp}/${archive}" -C "$tmp" || err "failed to extract ${archive}"
  [ -f "${tmp}/${BINARY}" ] || err "binary '${BINARY}' not found inside the archive"
  chmod +x "${tmp}/${BINARY}"

  # --- install (elevate only if necessary) ---------------------------------

  if [ -w "$INSTALL_DIR" ]; then
    mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  elif have sudo; then
    info "Elevated permissions needed to write to ${INSTALL_DIR}"
    sudo mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    err "cannot write to ${INSTALL_DIR} and sudo is unavailable; set FORGESYNC_INSTALL_DIR to a writable path"
  fi

  info "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
  info "Run '${BINARY} version' to verify."
}

main "$@"
