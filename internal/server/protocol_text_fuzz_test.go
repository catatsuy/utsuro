package server

import (
	"strings"
	"testing"
)

func FuzzParseLine(f *testing.F) {
	f.Add("get key\r\n")
	f.Add("SET A 1 0 3\n")
	f.Add("quit\r")
	f.Add("")
	f.Add("   ")

	f.Fuzz(func(t *testing.T, line string) {
		req, err := parseLine(line)
		if err != nil {
			return
		}

		if req.cmd != strings.ToLower(req.cmd) {
			t.Fatalf("command must be lowercase: %q", req.cmd)
		}
		if req.isQuit {
			if req.cmd != "quit" {
				t.Fatalf("quit request must have quit cmd: %q", req.cmd)
			}
			if len(req.args) != 0 {
				t.Fatalf("quit request must not have args: %v", req.args)
			}
		}
	})
}

func FuzzParseSetArgs(f *testing.F) {
	f.Add([]byte("k 0 0 1"))
	f.Add([]byte("k 12 10 0"))
	f.Add([]byte("k -1 0 1"))
	f.Add([]byte("k 0 0 -1"))
	f.Add([]byte("k"))

	f.Fuzz(func(t *testing.T, raw []byte) {
		args := strings.Fields(string(raw))
		key, _, bytesN, err := parseSetArgs(args)
		if err != nil {
			return
		}

		if key == "" {
			t.Fatal("key must not be empty when parse succeeds")
		}
		if bytesN < 0 {
			t.Fatalf("bytes must be non-negative: %d", bytesN)
		}
	})
}

func FuzzParseDeltaArgs(f *testing.F) {
	f.Add([]byte("k 1"))
	f.Add([]byte("k 0"))
	f.Add([]byte("k -1"))
	f.Add([]byte("only-key"))

	f.Fuzz(func(t *testing.T, raw []byte) {
		args := strings.Fields(string(raw))
		key, delta, err := parseDeltaArgs(args)
		if err != nil {
			return
		}

		if key == "" {
			t.Fatal("key must not be empty when parse succeeds")
		}
		_ = delta
	})
}
