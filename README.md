# utsuro

`utsuro` is an in-memory volatile KVS server that supports a subset of the memcached text protocol.

## Name

`utsuro` comes from the Japanese word *utsuro* (うつろ), meaning “empty”, “hollow”, or “transient”.
It reflects that this server is an in-memory volatile KVS: data may disappear at any time.

## Warning

- Data is volatile and may be lost at any time.
- Do not store data that must not disappear.

## MVP command subset

- `get`
- `gets`
- `set`
- `delete`
- `incr`
- `decr`

## Options

- `-listen` (default: `127.0.0.1:11211`)
- `-max-bytes` (default: `268435456`)
- `-target-bytes` (default: `max-bytes * 95 / 100`)
- `-evict-max` (default: `64`)
- `-incr-sliding-ttl-seconds` (default: `0`, disabled)
- `-verbose`

## Differences from memcached

- This server implements only a subset of memcached text protocol commands.
- `gets` is supported and returns the CAS token in `VALUE` response header.
- `cas` command is not implemented.
- `incr` on a missing key creates the key and returns `delta` (memcached returns `NOT_FOUND`).
- `decr` on a missing key creates the key with `0` and returns `0`.
- `decr` is clamped at `0`.
- Numeric values are treated as `uint64`.
- Non-numeric values return `CLIENT_ERROR cannot increment or decrement non-numeric value`.
- `incr` overflow (`uint64` max exceeded) returns `CLIENT_ERROR increment or decrement overflow`.
- When `-incr-sliding-ttl-seconds > 0`, successful `incr/decr` always set/update TTL to `now + ttl`.
- If a key is expired at `incr/decr` time, it is deleted first and treated as missing (then created by rules above).
