# lim - Light Image Monitoring

Minimal Go CLI skeleton.

## Run

```sh
go run .
```

Examples:

```sh
go run .              # monitors `docker events` (container create)
go run . list         # prints images + last-run timestamps
go run . list --before 24h  # only images last run > 24 hours ago
go run . help
```

## Build

```sh
go build -o lim .
./lim            # monitor
./lim list
```

## Distribute (Linux)

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
