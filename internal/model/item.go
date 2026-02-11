package model

type Item struct {
	Key   string
	Value []byte

	Flags uint32
	Size  int64

	// ExpUnix is Unix seconds. 0 means no expiration.
	ExpUnix int64
}
