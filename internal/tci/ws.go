// Package tci implements a client for Thetis's TCI-over-WebSocket protocol
// (Project Files/Source/Console/TCIServer.cs), which is protocol-compatible
// with ExpertSDR3/SunSDR2 PRO "TCI" (github.com/ExpertSDR3/TCI). Thetis's own
// server is a hand-rolled WebSocket implementation directly on TcpListener,
// not a standard library — this client hand-rolls the matching minimal
// RFC 6455 client subset (plain HTTP upgrade, masked/unmasked data frames,
// ping/pong/close) rather than pulling in a third-party WebSocket dependency.
package tci

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Opcodes per RFC 6455 §5.2.
const (
	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// Conn is a client-role WebSocket connection to a Thetis TCI server.
type Conn struct {
	tcp     net.Conn
	r       *bufio.Reader
	writeMu sync.Mutex
	timeout time.Duration
}

// Dial performs the WebSocket handshake against a Thetis TCI server at addr
// (host:port, e.g. "192.168.1.50:50001").
func Dial(addr string, timeout time.Duration) (*Conn, error) {
	tcpConn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("tci: dial %s: %w", addr, err)
	}
	if timeout > 0 {
		tcpConn.SetDeadline(time.Now().Add(timeout))
	}

	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("tci: generate key: %w", err)
	}
	encodedKey := base64.StdEncoding.EncodeToString(key)

	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}

	req := "GET / HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + encodedKey + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"

	if _, err := tcpConn.Write([]byte(req)); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("tci: send handshake: %w", err)
	}

	br := bufio.NewReader(tcpConn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("tci: read handshake response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		tcpConn.Close()
		return nil, fmt.Errorf("tci: handshake failed: %s", resp.Status)
	}
	if !strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") {
		tcpConn.Close()
		return nil, fmt.Errorf("tci: handshake missing Upgrade: websocket header")
	}

	want := acceptKey(encodedKey)
	got := resp.Header.Get("Sec-WebSocket-Accept")
	if got != want {
		tcpConn.Close()
		return nil, fmt.Errorf("tci: Sec-WebSocket-Accept mismatch: got %q want %q", got, want)
	}

	if timeout > 0 {
		tcpConn.SetDeadline(time.Time{})
	}

	return &Conn{tcp: tcpConn, r: br, timeout: timeout}, nil
}

func acceptKey(clientKey string) string {
	h := sha1.New()
	io.WriteString(h, clientKey+wsGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Close sends a close frame and closes the underlying TCP connection.
func (c *Conn) Close() error {
	_ = c.writeFrame(opClose, nil)
	return c.tcp.Close()
}

// WriteText sends a masked text frame — the TCI control-command channel.
func (c *Conn) WriteText(s string) error {
	return c.writeFrame(opText, []byte(s))
}

// WriteBinary sends a masked binary frame — the TCI audio-stream channel.
func (c *Conn) WriteBinary(b []byte) error {
	return c.writeFrame(opBinary, b)
}

func (c *Conn) writeFrame(opcode byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.timeout > 0 {
		c.tcp.SetWriteDeadline(time.Now().Add(c.timeout))
	}

	var header []byte
	header = append(header, 0x80|opcode) // FIN=1, opcode

	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return fmt.Errorf("tci: generate mask: %w", err)
	}

	n := len(payload)
	switch {
	case n <= 125:
		header = append(header, 0x80|byte(n))
	case n <= 0xFFFF:
		header = append(header, 0x80|126)
		var ext [2]byte
		binary.BigEndian.PutUint16(ext[:], uint16(n))
		header = append(header, ext[:]...)
	default:
		header = append(header, 0x80|127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		header = append(header, ext[:]...)
	}
	header = append(header, mask...)

	masked := make([]byte, n)
	for i, b := range payload {
		masked[i] = b ^ mask[i%4]
	}

	if _, err := c.tcp.Write(header); err != nil {
		return fmt.Errorf("tci: write frame header: %w", err)
	}
	if _, err := c.tcp.Write(masked); err != nil {
		return fmt.Errorf("tci: write frame payload: %w", err)
	}
	return nil
}

// ReadFrame reads one logical WebSocket message (reassembling fragmented
// frames) and returns its opcode and payload. Ping frames are answered with
// pong automatically and then the next frame is read; callers see only
// text/binary/close messages.
func (c *Conn) ReadFrame() (opcode byte, payload []byte, err error) {
	for {
		op, fin, data, rerr := c.readRawFrame()
		if rerr != nil {
			return 0, nil, rerr
		}

		switch op {
		case opPing:
			if werr := c.writeFrame(opPong, data); werr != nil {
				return 0, nil, werr
			}
			continue
		case opPong:
			continue
		case opClose:
			return opClose, data, nil
		}

		if opcode == 0 && op != opContinuation {
			opcode = op
		}
		payload = append(payload, data...)
		if fin {
			return opcode, payload, nil
		}
	}
}

func (c *Conn) readRawFrame() (opcode byte, fin bool, payload []byte, err error) {
	if c.timeout > 0 {
		c.tcp.SetReadDeadline(time.Now().Add(c.timeout))
	}

	var head [2]byte
	if _, err = io.ReadFull(c.r, head[:]); err != nil {
		return 0, false, nil, fmt.Errorf("tci: read frame header: %w", err)
	}

	fin = head[0]&0x80 != 0
	opcode = head[0] & 0x0F
	masked := head[1]&0x80 != 0
	length := uint64(head[1] & 0x7F)

	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(c.r, ext[:]); err != nil {
			return 0, false, nil, fmt.Errorf("tci: read extended length: %w", err)
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(c.r, ext[:]); err != nil {
			return 0, false, nil, fmt.Errorf("tci: read extended length: %w", err)
		}
		length = binary.BigEndian.Uint64(ext[:])
	}

	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(c.r, maskKey[:]); err != nil {
			return 0, false, nil, fmt.Errorf("tci: read mask key: %w", err)
		}
	}

	payload = make([]byte, length)
	if _, err = io.ReadFull(c.r, payload); err != nil {
		return 0, false, nil, fmt.Errorf("tci: read payload: %w", err)
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, fin, payload, nil
}
