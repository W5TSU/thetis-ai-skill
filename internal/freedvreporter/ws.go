// Package freedvreporter is a client for FreeDV Reporter (qso.freedv.org)'s
// live activity feed, used to watch for real off-air FreeDV transmissions
// worth tuning to (see Documentation/FreeDV-Plan.md, Stage D).
//
// The reporter has no REST/JSON polling API — its live data arrives over a
// Socket.IO v4 connection (confirmed against the site's own
// /static/js/index.js and by direct protocol probing, 2026-08-09). Matching
// this project's existing internal/tci package, this hand-rolls the minimal
// RFC 6455 client subset directly (no third-party WebSocket dependency) —
// the one addition over internal/tci's version is TLS, since the reporter is
// wss:// while Thetis's own TCI server is plain ws:// on a LAN.
package freedvreporter

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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

// wsConn is a client-role TLS WebSocket connection.
type wsConn struct {
	tcp     net.Conn
	r       *bufio.Reader
	timeout time.Duration
}

// dialWS performs a TLS + WebSocket handshake against host:443, requesting
// path (which must include any query string, e.g. "/socket.io/?EIO=4&...").
func dialWS(host, path string, timeout time.Duration) (*wsConn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, "443"), &tls.Config{ServerName: host})
	if err != nil {
		return nil, fmt.Errorf("freedvreporter: tls dial %s: %w", host, err)
	}
	if timeout > 0 {
		tlsConn.SetDeadline(time.Now().Add(timeout))
	}

	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("freedvreporter: generate key: %w", err)
	}
	encodedKey := base64.StdEncoding.EncodeToString(key)

	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + encodedKey + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"

	if _, err := tlsConn.Write([]byte(req)); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("freedvreporter: send handshake: %w", err)
	}

	br := bufio.NewReader(tlsConn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("freedvreporter: read handshake response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		tlsConn.Close()
		return nil, fmt.Errorf("freedvreporter: handshake failed: %s", resp.Status)
	}
	if !strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") {
		tlsConn.Close()
		return nil, fmt.Errorf("freedvreporter: handshake missing Upgrade: websocket header")
	}
	if got, want := resp.Header.Get("Sec-WebSocket-Accept"), acceptKey(encodedKey); got != want {
		tlsConn.Close()
		return nil, fmt.Errorf("freedvreporter: Sec-WebSocket-Accept mismatch: got %q want %q", got, want)
	}

	if timeout > 0 {
		tlsConn.SetDeadline(time.Time{})
	}

	return &wsConn{tcp: tlsConn, r: br, timeout: timeout}, nil
}

func acceptKey(clientKey string) string {
	h := sha1.New()
	io.WriteString(h, clientKey+wsGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (c *wsConn) Close() error {
	_ = c.writeFrame(opClose, nil)
	return c.tcp.Close()
}

// WriteText sends a masked text frame (client->server frames must be masked
// per RFC 6455 §5.1).
func (c *wsConn) WriteText(s string) error {
	return c.writeFrame(opText, []byte(s))
}

func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	if c.timeout > 0 {
		c.tcp.SetWriteDeadline(time.Now().Add(c.timeout))
	}

	var header []byte
	header = append(header, 0x80|opcode) // FIN=1, opcode

	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return fmt.Errorf("freedvreporter: generate mask: %w", err)
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
		return fmt.Errorf("freedvreporter: write frame header: %w", err)
	}
	if _, err := c.tcp.Write(masked); err != nil {
		return fmt.Errorf("freedvreporter: write frame payload: %w", err)
	}
	return nil
}

// ReadFrame reads one logical WebSocket message (reassembling fragmented
// frames), transparently answering pings with pongs. Callers see only
// text/binary/close messages.
func (c *wsConn) ReadFrame() (opcode byte, payload []byte, err error) {
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

func (c *wsConn) readRawFrame() (opcode byte, fin bool, payload []byte, err error) {
	if c.timeout > 0 {
		c.tcp.SetReadDeadline(time.Now().Add(c.timeout))
	}

	var head [2]byte
	if _, err = io.ReadFull(c.r, head[:]); err != nil {
		return 0, false, nil, fmt.Errorf("freedvreporter: read frame header: %w", err)
	}

	fin = head[0]&0x80 != 0
	opcode = head[0] & 0x0F
	masked := head[1]&0x80 != 0
	length := uint64(head[1] & 0x7F)

	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(c.r, ext[:]); err != nil {
			return 0, false, nil, fmt.Errorf("freedvreporter: read extended length: %w", err)
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(c.r, ext[:]); err != nil {
			return 0, false, nil, fmt.Errorf("freedvreporter: read extended length: %w", err)
		}
		length = binary.BigEndian.Uint64(ext[:])
	}

	// Server->client frames are never masked per RFC 6455 §5.1, but a mask
	// key is still read/applied if the server sets the bit anyway, matching
	// internal/tci/ws.go's defensive handling.
	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(c.r, maskKey[:]); err != nil {
			return 0, false, nil, fmt.Errorf("freedvreporter: read mask key: %w", err)
		}
	}

	payload = make([]byte, length)
	if _, err = io.ReadFull(c.r, payload); err != nil {
		return 0, false, nil, fmt.Errorf("freedvreporter: read payload: %w", err)
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, fin, payload, nil
}
