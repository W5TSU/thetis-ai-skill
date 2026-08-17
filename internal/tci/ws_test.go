package tci

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

// TestAcceptKeyRFC6455Vector checks acceptKey against the worked example in
// RFC 6455 §1.3.
func TestAcceptKeyRFC6455Vector(t *testing.T) {
	got := acceptKey("dGhlIHNhbXBsZSBub25jZQ==")
	want := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if got != want {
		t.Errorf("acceptKey = %q, want %q", got, want)
	}
}

// TestFrameRoundTrip writes masked text/binary frames on one end of a
// net.Pipe and reads them back on the other, verifying the client's own
// frame codec is self-consistent (readRawFrame can decode writeFrame's
// output, matching what a spec-compliant server would send/receive).
func TestFrameRoundTrip(t *testing.T) {
	clientEnd, serverEnd := net.Pipe()
	defer clientEnd.Close()
	defer serverEnd.Close()

	c := &Conn{tcp: clientEnd, r: bufio.NewReader(clientEnd), timeout: 2 * time.Second}
	s := &Conn{tcp: serverEnd, r: bufio.NewReader(serverEnd), timeout: 2 * time.Second}

	done := make(chan error, 1)
	go func() {
		op, payload, err := s.ReadFrame()
		if err != nil {
			done <- err
			return
		}
		if op != opText || string(payload) != "hello;" {
			done <- fmt.Errorf("got opcode %d payload %q, want text \"hello;\"", op, payload)
			return
		}
		done <- nil
	}()

	if err := c.WriteText("hello;"); err != nil {
		t.Fatalf("WriteText: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// Large binary payload exercises the 16-bit extended-length path.
	big := bytes.Repeat([]byte{0xAB}, 70000)
	go func() {
		op, payload, err := s.ReadFrame()
		if err != nil {
			done <- err
			return
		}
		if op != opBinary || !bytes.Equal(payload, big) {
			done <- fmt.Errorf("binary frame mismatch: opcode %d len %d, want %d", op, len(payload), len(big))
			return
		}
		done <- nil
	}()
	if err := c.WriteBinary(big); err != nil {
		t.Fatalf("WriteBinary: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
