# benchmark module

This directory is a separate Go module for `utsuro`.

## Run test

```bash
cd benchmark
go test -v
```

What it does:

- Starts `utsuro` server in-process (`internal/server`)
- Uses `github.com/bradfitz/gomemcache/memcache`
- Verifies `utsuro`-compatible command behavior
