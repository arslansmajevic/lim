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
