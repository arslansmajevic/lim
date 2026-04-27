# lim - Light Image Monitoring

Minimal Go CLI skeleton.

## Run

```sh
go run .
```

Examples:

```sh
go run .              # shows status and ensures monitoring is running in background
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

1) Download the right binary for your machine (amd64 vs arm64) from the GitHub Release assets.

Example (amd64) — replace `<version>` (e.g. `v0.1.0`) and `<owner>/<repo>`:

```sh
VERSION=<version>
REPO=<owner>/<repo>

curl -fsSL -o lim "https://github.com/${REPO}/releases/download/${VERSION}/lim-linux-amd64"
curl -fsSL -o lim.sha256 "https://github.com/${REPO}/releases/download/${VERSION}/lim-linux-amd64.sha256"

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
