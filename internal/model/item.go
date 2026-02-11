package model

type Item struct {
	Key   string
	Value []byte

	Flags uint32
	Size  int64
}
