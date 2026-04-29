#!/usr/bin/env sh
set -eu

REPO="${REPO:-arslansmajevic/lim}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="lim"
SYSTEMD_SERVICE="${SYSTEMD_SERVICE:-1}"
SERVICE_USER="${SERVICE_USER:-}"
SERVICE_GROUP="${SERVICE_GROUP:-}"
STATE_DIR="${STATE_DIR:-/var/lib/lim}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

download() {
  url="$1"
  out="$2"

  if need_cmd curl; then
    curl -fsSL -o "$out" "$url"
    return 0
  fi

  if need_cmd wget; then
    wget -qO "$out" "$url"
    return 0
  fi

  echo "error: need curl or wget" >&2
  return 1
}

sha256_check() {
  file="$1"
  sha_file="$2"

  if need_cmd sha256sum; then
    (cd "$(dirname "$sha_file")" && sha256sum -c "$(basename "$sha_file")" >/dev/null)
    return 0
  fi

  if need_cmd shasum; then
    expected="$(awk '{print $1}' "$sha_file")"
    actual="$(shasum -a 256 "$file" | awk '{print $1}')"
    if [ "$expected" = "$actual" ]; then
      return 0
    fi
    echo "error: sha256 mismatch" >&2
    return 1
  fi

  echo "error: need sha256sum (preferred) or shasum" >&2
  return 1
}

OS="$(uname -s)"
ARCH="$(uname -m)"

if [ "$OS" != "Linux" ]; then
  echo "error: this installer currently supports Linux only (got $OS)" >&2
  exit 1
fi

case "$ARCH" in
  x86_64|amd64)
    ARCH=amd64
    ;;
  aarch64|arm64)
    ARCH=arm64
    ;;
  *)
    echo "error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
 esac

ASSET="lim-linux-$ARCH"
BASE_URL="https://github.com/${REPO}/releases/latest/download"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

BIN_PATH="$TMP_DIR/$ASSET"
SHA_PATH="$TMP_DIR/$ASSET.sha256"

echo "Downloading ${REPO} latest release (${ASSET})..." >&2

download "$BASE_URL/$ASSET" "$BIN_PATH"
download "$BASE_URL/$ASSET.sha256" "$SHA_PATH"

echo "Verifying checksum..." >&2
sha256_check "$TMP_DIR/$ASSET" "$SHA_PATH"

chmod +x "$TMP_DIR/$ASSET"

if [ "$(id -u)" -ne 0 ]; then
  if need_cmd sudo; then
    SUDO="sudo"
  else
    # If the install dir is user-writable, allow installing without sudo.
    if [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
      SUDO=""
    else
      if mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ]; then
        SUDO=""
      else
        echo "error: need sudo (or run as root) to install into $INSTALL_DIR" >&2
        exit 1
      fi
    fi
  fi
else
  SUDO=""
fi

$SUDO install -m 0755 "$TMP_DIR/$ASSET" "$INSTALL_DIR/$BINARY_NAME"

echo "Installed $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME" >&2

# Optional: install a systemd service that starts on boot (after Docker).
if [ "$SYSTEMD_SERVICE" = "1" ] && need_cmd systemctl && [ -d /run/systemd/system ]; then
  echo "Installing systemd service lim.service..." >&2

  if [ "$(id -u)" -ne 0 ] && [ -z "$SUDO" ]; then
    echo "warning: skipping systemd service install (need sudo/root)" >&2
    echo "Run: $BINARY_NAME" >&2
    exit 0
  fi

  if [ -n "$SERVICE_USER" ]; then
    if ! id "$SERVICE_USER" >/dev/null 2>&1; then
      echo "error: SERVICE_USER '$SERVICE_USER' does not exist" >&2
      exit 1
    fi
    if [ -z "$SERVICE_GROUP" ]; then
      # Prefer the user's primary group.
      SERVICE_GROUP="$(id -gn "$SERVICE_USER" 2>/dev/null || echo "")"
    fi
    if [ -z "$SERVICE_GROUP" ]; then
      SERVICE_GROUP="$SERVICE_USER"
    fi
  fi

  # Create/prepare a shared state directory for the service.
  $SUDO mkdir -p "$STATE_DIR"
  $SUDO chmod 0755 "$STATE_DIR"
  if [ -n "$SERVICE_USER" ]; then
    $SUDO chown "$SERVICE_USER:$SERVICE_GROUP" "$STATE_DIR"
  fi

  UNIT_PATH="/etc/systemd/system/lim.service"
  $SUDO sh -c "cat > '$UNIT_PATH'" <<EOF
[Unit]
Description=lim (Light Image Monitoring)
Wants=docker.service
After=docker.service docker.socket network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=LIM_STATE_DIR=$STATE_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME _monitor
$( [ -n "$SERVICE_USER" ] && printf 'User=%s\nGroup=%s\n' "$SERVICE_USER" "$SERVICE_GROUP" )
Restart=on-failure
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF

  $SUDO systemctl daemon-reload
  $SUDO systemctl enable --now lim.service

  echo "Enabled lim.service (starts on boot)." >&2
  echo "Check: systemctl status lim.service" >&2
else
  echo "Run: $BINARY_NAME" >&2
fi
