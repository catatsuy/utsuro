package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/catatsuy/utsuro/internal/cache"
)

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	for {
		line, err := readCommandLine(r)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.logf("read error: %v", err)
			}
			return
		}

		req, err := parseLine(line)
		if err != nil {
			_ = writeClientError(w, "bad command line format")
			if flushErr := w.Flush(); flushErr != nil {
				return
			}
			continue
		}
		if req.isQuit {
			return
		}

		switch req.cmd {
		case "get":
			err = s.handleGetLike(w, req.args, false)
		case "gets":
			err = s.handleGetLike(w, req.args, true)
		case "set":
			err = s.handleSet(r, w, req.args)
		case "delete":
			err = s.handleDelete(w, req.args)
		case "incr":
			err = s.handleIncrDecr(w, req.args, true)
		case "decr":
			err = s.handleIncrDecr(w, req.args, false)
		default:
			err = writeClientError(w, "unknown command")
		}
		if err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
	}
}

func (s *Server) handleGetLike(w *bufio.Writer, args []string, withCAS bool) error {
	if len(args) == 0 {
		return writeClientError(w, "get requires at least one key")
	}

	for _, key := range args {
		item, ok := s.cache.Get(key)
		if !ok {
			continue
		}
		if withCAS {
			if _, err := fmt.Fprintf(w, "VALUE %s %d %d %d\r\n", key, item.Flags, len(item.Value), item.CAS); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "VALUE %s %d %d\r\n", key, item.Flags, len(item.Value)); err != nil {
				return err
			}
		}
		if _, err := w.Write(item.Value); err != nil {
			return err
		}
		if _, err := w.WriteString("\r\n"); err != nil {
			return err
		}
	}
	_, err := w.WriteString("END\r\n")
	return err
}

func (s *Server) handleSet(r *bufio.Reader, w *bufio.Writer, args []string) error {
	key, flags, bytesN, err := parseSetArgs(args)
	if err != nil {
		return writeClientError(w, err.Error())
	}

	value := make([]byte, bytesN)
	if _, err := io.ReadFull(r, value); err != nil {
		return writeClientError(w, "bad data chunk")
	}
	if err := consumeChunkTerminator(r); err != nil {
		return writeClientError(w, "bad data chunk")
	}

	if err := s.cache.Set(key, flags, value); err != nil {
		if errors.Is(err, cache.ErrObjectTooLarge) || errors.Is(err, cache.ErrNoSpace) {
			return writeServerError(w, err.Error())
		}
		return writeServerError(w, "internal error")
	}

	_, err = w.WriteString("STORED\r\n")
	return err
}

func (s *Server) handleDelete(w *bufio.Writer, args []string) error {
	if len(args) != 1 {
		return writeClientError(w, "delete requires key")
	}
	if s.cache.Delete(args[0]) {
		_, err := w.WriteString("DELETED\r\n")
		return err
	}
	_, err := w.WriteString("NOT_FOUND\r\n")
	return err
}

func (s *Server) handleIncrDecr(w *bufio.Writer, args []string, incr bool) error {
	key, delta, err := parseDeltaArgs(args)
	if err != nil {
		return writeClientError(w, err.Error())
	}

	var value uint64
	if incr {
		value, err = s.cache.Incr(key, delta)
	} else {
		value, err = s.cache.Decr(key, delta)
	}
	if err != nil {
		if errors.Is(err, cache.ErrNonNumeric) {
			return writeClientError(w, cache.ErrNonNumeric.Error())
		}
		if errors.Is(err, cache.ErrOverflow) {
			return writeClientError(w, cache.ErrOverflow.Error())
		}
		if errors.Is(err, cache.ErrObjectTooLarge) || errors.Is(err, cache.ErrNoSpace) {
			return writeServerError(w, err.Error())
		}
		return writeServerError(w, "internal error")
	}

	_, err = fmt.Fprintf(w, "%d\r\n", value)
	return err
}

func writeClientError(w *bufio.Writer, msg string) error {
	_, err := fmt.Fprintf(w, "CLIENT_ERROR %s\r\n", msg)
	return err
}

func writeServerError(w *bufio.Writer, msg string) error {
	_, err := fmt.Fprintf(w, "SERVER_ERROR %s\r\n", msg)
	return err
}

// readCommandLine accepts CRLF, LF, CR and CR NUL (common telnet newline).
func readCommandLine(r *bufio.Reader) (string, error) {
	var buf bytes.Buffer

	for {
		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && buf.Len() > 0 {
				return buf.String(), nil
			}
			return "", err
		}

		switch b {
		case '\n':
			return buf.String(), nil
		case '\r':
			next, err := r.ReadByte()
			if err == nil {
				if next != '\n' && next != 0x00 {
					if unreadErr := r.UnreadByte(); unreadErr != nil {
						return "", unreadErr
					}
				}
			} else if !errors.Is(err, io.EOF) {
				return "", err
			}
			return buf.String(), nil
		default:
			buf.WriteByte(b)
		}
	}
}

// consumeChunkTerminator accepts CRLF, LF, CR and CR NUL after set payload.
func consumeChunkTerminator(r *bufio.Reader) error {
	b, err := r.ReadByte()
	if err != nil {
		return err
	}
	switch b {
	case '\n':
		return nil
	case '\r':
		next, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if next == '\n' || next == 0x00 {
			return nil
		}
		return fmt.Errorf("invalid chunk terminator")
	default:
		return fmt.Errorf("invalid chunk terminator")
	}
}
