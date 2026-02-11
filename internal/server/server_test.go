package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
)

func newPipeSession(t *testing.T) (net.Conn, func()) {
	t.Helper()

	srv := NewServer(Config{
		MaxBytes:      1 << 20,
		TargetBytes:   (1 << 20) * 95 / 100,
		MaxEvictPerOp: 64,
	})

	serverSide, clientSide := net.Pipe()
	go srv.handleConn(serverSide)

	return clientSide, func() {
		_ = clientSide.Close()
	}
}

func sendCommand(t *testing.T, conn net.Conn, cmd string, readUntil string) string {
	t.Helper()
	if _, err := conn.Write([]byte(cmd)); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	r := bufio.NewReader(conn)
	var b strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		b.WriteString(line)
		if strings.HasSuffix(b.String(), readUntil) {
			return b.String()
		}
	}
}

func TestSetGetDelete(t *testing.T) {
	conn, stop := newPipeSession(t)
	defer stop()

	resp := sendCommand(t, conn, "set a 12 0 3\r\nfoo\r\n", "\r\n")
	if resp != "STORED\r\n" {
		t.Fatalf("unexpected set response: %q", resp)
	}

	resp = sendCommand(t, conn, "get a\r\n", "END\r\n")
	expected := "VALUE a 12 3\r\nfoo\r\nEND\r\n"
	if resp != expected {
		t.Fatalf("unexpected get response:\nwant=%q\n got=%q", expected, resp)
	}

	resp = sendCommand(t, conn, "delete a\r\n", "\r\n")
	if resp != "DELETED\r\n" {
		t.Fatalf("unexpected delete response: %q", resp)
	}

	resp = sendCommand(t, conn, "get a\r\n", "END\r\n")
	if resp != "END\r\n" {
		t.Fatalf("unexpected get after delete response: %q", resp)
	}
}

func TestIncr(t *testing.T) {
	conn, stop := newPipeSession(t)
	defer stop()

	resp := sendCommand(t, conn, "incr cnt 5\r\n", "\r\n")
	if resp != "5\r\n" {
		t.Fatalf("unexpected incr missing response: %q", resp)
	}

	resp = sendCommand(t, conn, "incr cnt 7\r\n", "\r\n")
	if resp != "12\r\n" {
		t.Fatalf("unexpected incr existing response: %q", resp)
	}

	resp = sendCommand(t, conn, "set s 0 0 3\r\nabc\r\n", "\r\n")
	if resp != "STORED\r\n" {
		t.Fatalf("unexpected set response: %q", resp)
	}
	resp = sendCommand(t, conn, "incr s 1\r\n", "\r\n")
	if resp != "CLIENT_ERROR cannot increment or decrement non-numeric value\r\n" {
		t.Fatalf("unexpected non numeric response: %q", resp)
	}

	resp = sendCommand(t, conn, "set max 0 0 20\r\n18446744073709551615\r\n", "\r\n")
	if resp != "STORED\r\n" {
		t.Fatalf("unexpected set max response: %q", resp)
	}
	resp = sendCommand(t, conn, "incr max 1\r\n", "\r\n")
	if resp != "CLIENT_ERROR increment or decrement overflow\r\n" {
		t.Fatalf("unexpected overflow response: %q", resp)
	}
}

func TestDecr(t *testing.T) {
	conn, stop := newPipeSession(t)
	defer stop()

	resp := sendCommand(t, conn, "decr missing 9\r\n", "\r\n")
	if resp != "0\r\n" {
		t.Fatalf("unexpected decr missing response: %q", resp)
	}

	resp = sendCommand(t, conn, "set n 0 0 1\r\n1\r\n", "\r\n")
	if resp != "STORED\r\n" {
		t.Fatalf("unexpected set response: %q", resp)
	}

	resp = sendCommand(t, conn, "decr n 2\r\n", "\r\n")
	if resp != "0\r\n" {
		t.Fatalf("unexpected decr clamp response: %q", resp)
	}
}

func TestMultiGetAndBadDataChunk(t *testing.T) {
	conn, stop := newPipeSession(t)
	defer stop()

	for i := 1; i <= 2; i++ {
		resp := sendCommand(t, conn, fmt.Sprintf("set k%d 0 0 2\r\nv%d\r\n", i, i), "\r\n")
		if resp != "STORED\r\n" {
			t.Fatalf("unexpected set response: %q", resp)
		}
	}

	resp := sendCommand(t, conn, "get k1 k2 missing\r\n", "END\r\n")
	if !strings.Contains(resp, "VALUE k1 0 2\r\nv1\r\n") {
		t.Fatalf("missing k1 response: %q", resp)
	}
	if !strings.Contains(resp, "VALUE k2 0 2\r\nv2\r\n") {
		t.Fatalf("missing k2 response: %q", resp)
	}
	if !strings.HasSuffix(resp, "END\r\n") {
		t.Fatalf("missing END terminator: %q", resp)
	}

	resp = sendCommand(t, conn, "set bad 0 0 3\r\nabcX", "\r\n")
	if resp != "CLIENT_ERROR bad data chunk\r\n" {
		t.Fatalf("unexpected bad chunk response: %q", resp)
	}
}
