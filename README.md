# lim - Light Image Monitoring

Minimal Go CLI skeleton.

## Run

```sh
go run .
```

Examples:

```sh
go run .              # shows status and ensures monitoring is running in background
go run . --status     # prints status only
go run . list         # prints images + last-run timestamps
go run . list --before 24h  # only images last run > 24 hours ago
go run . --shutdown   # stop background monitor
go run . help
```

## Build

```sh
go build -o lim .
./lim            # shows status and starts/ensures background monitor
./lim list
./lim --shutdown
```

## Distribute (Linux)

### Download from GitHub Releases

One-liner install (downloads latest release and installs to `/usr/local/bin/lim`):

```sh
curl -fsSL https://raw.githubusercontent.com/arslansmajevic/lim/main/install.sh | sh
```

One-liner uninstall:

```sh
curl -fsSL https://raw.githubusercontent.com/arslansmajevic/lim/main/uninstall.sh | sh
```

Optional uninstall + purge local config/state:

```sh
curl -fsSL https://raw.githubusercontent.com/arslansmajevic/lim/main/uninstall.sh | PURGE_CONFIG=1 sh
```

1) Download the right binary for your machine (amd64 vs arm64) from the GitHub Release assets.

Example (amd64) — downloads from the latest GitHub Release:

```sh
REPO=arslansmajevic/lim

curl -fsSL -o lim "https://github.com/${REPO}/releases/latest/download/lim-linux-amd64"
curl -fsSL -o lim.sha256 "https://github.com/${REPO}/releases/latest/download/lim-linux-amd64.sha256"

sha256sum -c lim.sha256

chmod +x lim
sudo install -m 0755 lim /usr/local/bin/lim
```

2) Run it:

```sh
lim
```

### Build yourself

Build static Linux binaries:

```sh
make dist-linux
ls -lah dist/
```

Install on a Linux machine so `lim` is runnable from anywhere:

```sh
sudo install -m 0755 ./dist/lim-linux-amd64 /usr/local/bin/lim
lim
```

Notes:

- `lim` requires Docker to be installed and the daemon reachable; otherwise it exits with an error.
- Only one `lim` monitor instance runs at a time; re-running `lim` prints "monitor already running".
