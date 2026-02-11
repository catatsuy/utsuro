# utsuro

`utsuro` is an in-memory volatile KVS server that supports a subset of the memcached text protocol.

## Warning

- Data is volatile and may be lost at any time.
- Do not store data that must not disappear.

## MVP command subset

- `get`
- `set`
- `delete`
- `incr`
- `decr`

## Compatibility notes

- This is a subset implementation of the memcached text protocol.
- `incr` behavior is intentionally incompatible for missing keys: it creates the key and returns the value.
- `decr` on missing keys creates `0` and returns `0`.
