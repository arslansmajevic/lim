#!/usr/bin/env sh
set -eu

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="lim"
SYSTEMD_SERVICE="${SYSTEMD_SERVICE:-1}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

BIN_PATH="$INSTALL_DIR/$BINARY_NAME"

if [ "$SYSTEMD_SERVICE" = "1" ] && need_cmd systemctl && [ -d /run/systemd/system ]; then
  if [ "$(id -u)" -ne 0 ]; then
    if need_cmd sudo; then
      SUDO="sudo"
    else
      echo "error: need sudo (or run as root) to uninstall systemd service" >&2
      exit 1
    fi
  else
    SUDO=""
  fi

  if $SUDO systemctl is-enabled --quiet lim.service 2>/dev/null; then
    $SUDO systemctl disable --now lim.service || true
    echo "Disabled lim.service" >&2
  fi

  UNIT_PATH="/etc/systemd/system/lim.service"
  if [ -f "$UNIT_PATH" ]; then
    $SUDO rm -f "$UNIT_PATH"
    $SUDO systemctl daemon-reload || true
    echo "Removed $UNIT_PATH" >&2
  fi
fi

if [ ! -e "$BIN_PATH" ]; then
  echo "lim is not installed at $BIN_PATH" >&2
else
  if [ "$(id -u)" -ne 0 ]; then
    if need_cmd sudo; then
      SUDO="sudo"
    else
      echo "error: need sudo (or run as root) to remove $BIN_PATH" >&2
      exit 1
    fi
  else
    SUDO=""
  fi

  $SUDO rm -f "$BIN_PATH"
  echo "Removed $BIN_PATH" >&2
fi

# Optional: purge local config/state directory.
# Set PURGE_CONFIG=1 to remove it.
if [ "${PURGE_CONFIG:-0}" = "1" ]; then
  if [ -n "${XDG_CONFIG_HOME:-}" ]; then
    CONFIG_DIR="$XDG_CONFIG_HOME/lim"
  else
    CONFIG_DIR="$HOME/.config/lim"
  fi

  if [ -d "$CONFIG_DIR" ]; then
    rm -rf "$CONFIG_DIR"
    echo "Purged $CONFIG_DIR" >&2
  else
    echo "No config dir at $CONFIG_DIR" >&2
  fi
fi

# Optional: purge system-wide state directory used by the systemd service.
if [ "${PURGE_CONFIG:-0}" = "1" ] && [ -d /var/lib/lim ]; then
  rm -rf /var/lib/lim
  echo "Purged /var/lib/lim" >&2
fi
