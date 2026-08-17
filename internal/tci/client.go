package tci

import (
	"fmt"
	"strings"
)

// Client wraps a Conn with the TCI text control-command protocol: semicolon-
// terminated "cmd:arg,arg,...;" messages (not JSON, despite Thetis importing
// Newtonsoft.Json — see TCIServer.cs's big command switch ~1653-1673, 5294-5598).
type Client struct {
	conn *Conn
}

// NewClient wraps an already-dialed Conn.
func NewClient(conn *Conn) *Client {
	return &Client{conn: conn}
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// SendCmd writes "cmd:arg,arg,...;" as a text frame.
func (c *Client) SendCmd(cmd string, args ...string) error {
	msg := cmd + ":" + strings.Join(args, ",") + ";"
	return c.conn.WriteText(msg)
}

// SendBareCmd writes "cmd;" (no colon, no args) as a text frame. Thetis's
// parser (parseTextFrame, TCIServer.cs:5258-5285) splits each message on the
// first ':'; messages with no colon go to a separate dispatch table for
// argument-less queries/actions. A handful of commands — notably
// "cw_macros_stop" — exist ONLY in that colon-less table (confirmed against
// TCIServer.cs: "cw_macros_stop" appears once, in the bare-command switch,
// never in the "cmd:args;" switch) and are silently ignored if sent through
// SendCmd's "cmd:;" form.
func (c *Client) SendBareCmd(cmd string) error {
	return c.conn.WriteText(cmd + ";")
}

// RecvCmd reads the next text frame and splits it into command + args.
func (c *Client) RecvCmd() (cmd string, args []string, err error) {
	for {
		op, payload, err := c.conn.ReadFrame()
		if err != nil {
			return "", nil, err
		}
		if op == opClose {
			return "", nil, fmt.Errorf("tci: connection closed by server")
		}
		if op != opText {
			continue // audio/binary frames are read via RecvAudioFrame instead
		}
		return parseCmd(string(payload))
	}
}

func parseCmd(msg string) (cmd string, args []string, err error) {
	msg = strings.TrimSuffix(strings.TrimSpace(msg), ";")
	idx := strings.IndexByte(msg, ':')
	if idx < 0 {
		return msg, nil, nil
	}
	cmd = msg[:idx]
	rest := msg[idx+1:]
	if rest == "" {
		return cmd, nil, nil
	}
	return cmd, strings.Split(rest, ","), nil
}

// RecvAudioFrame reads the next binary frame as a raw TCI stream frame
// (header + samples) — see audio.go for the layout.
func (c *Client) RecvAudioFrame() (StreamHeader, []byte, error) {
	for {
		op, payload, err := c.conn.ReadFrame()
		if err != nil {
			return StreamHeader{}, nil, err
		}
		if op == opClose {
			return StreamHeader{}, nil, fmt.Errorf("tci: connection closed by server")
		}
		if op != opBinary {
			continue
		}
		return ParseStreamFrame(payload)
	}
}

// SendAudioFrame writes a raw TCI stream frame (header + samples) as a
// binary frame.
func (c *Client) SendAudioFrame(h StreamHeader, samples []byte) error {
	return c.conn.WriteBinary(BuildStreamFrame(h, samples))
}
