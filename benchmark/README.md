# benchmark module

This directory is a separate Go module for `utsuro`.

## Run demo

```bash
cd benchmark
go run .
```

What it does:

- Starts `utsuro` server in-process (`internal/server`)
- Uses `github.com/bradfitz/gomemcache/memcache`
- Executes `set/get/incr/decr`
- Prints results, including missing-key `incr` behavior

## Run test

```bash
cd benchmark
go test -v
```
