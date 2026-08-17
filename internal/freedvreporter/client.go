package freedvreporter

import (
	"encoding/json"
	"fmt"
	"time"
)

// ReporterHost is qso.freedv.org's fixed hostname. Exported as a const
// (rather than hardcoded in cmd/) since it's a property of the protocol
// this package speaks, not something a caller should need to override.
const ReporterHost = "qso.freedv.org"

const socketIOPath = "/socket.io/?EIO=4&transport=websocket"

// Client is a connected, authenticated ("view" role) Socket.IO v4 session
// against FreeDV Reporter's live feed.
type Client struct {
	ws *wsConn
}

// Dial connects to FreeDV Reporter and completes both the Engine.IO and
// Socket.IO v4 handshakes. Protocol confirmed by direct probing 2026-08-09:
// the server accepts a direct WebSocket connection (no polling transport
// needed first) at ReporterHost's "/socket.io/?EIO=4&transport=websocket",
// immediately sends an Engine.IO OPEN packet ("0{...}"), and expects a
// Socket.IO CONNECT packet ("40" + JSON auth) in reply before it will start
// pushing station/activity events.
func Dial(timeout time.Duration) (*Client, error) {
	ws, err := dialWS(ReporterHost, socketIOPath, timeout)
	if err != nil {
		return nil, err
	}

	// Engine.IO OPEN packet: "0{"sid":"...","pingInterval":...,...}". Not
	// otherwise needed (this client answers pings reactively rather than
	// tracking the advertised interval), but must be consumed before the
	// connection is usable.
	op, payload, err := ws.ReadFrame()
	if err != nil {
		ws.Close()
		return nil, fmt.Errorf("freedvreporter: read engine.io open packet: %w", err)
	}
	if op != opText || len(payload) == 0 || payload[0] != '0' {
		ws.Close()
		return nil, fmt.Errorf("freedvreporter: expected engine.io OPEN packet, got %q", truncate(payload, 80))
	}

	// Socket.IO CONNECT to the default "/" namespace. The site's own client
	// (index.js) sends { role: "view", protocol_version: 2 } as the `auth`
	// option — matched exactly here since an unrecognised auth shape may be
	// rejected server-side.
	auth, err := json.Marshal(struct {
		Role            string `json:"role"`
		ProtocolVersion int    `json:"protocol_version"`
	}{Role: "view", ProtocolVersion: 2})
	if err != nil {
		ws.Close()
		return nil, fmt.Errorf("freedvreporter: marshal auth: %w", err)
	}
	if err := ws.WriteText("40" + string(auth)); err != nil {
		ws.Close()
		return nil, fmt.Errorf("freedvreporter: send socket.io connect: %w", err)
	}

	// Socket.IO CONNECT ack: "40{"sid":"..."}" (a different sid than the
	// engine.io one above — this is the socket.io-level session).
	op, payload, err = ws.ReadFrame()
	if err != nil {
		ws.Close()
		return nil, fmt.Errorf("freedvreporter: read socket.io connect ack: %w", err)
	}
	if op != opText || len(payload) < 2 || string(payload[:2]) != "40" {
		ws.Close()
		return nil, fmt.Errorf("freedvreporter: expected socket.io CONNECT ack, got %q", truncate(payload, 80))
	}

	return &Client{ws: ws}, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.ws.Close()
}

// Event is one (event name, raw JSON payload) pair as sent by the server.
// For "bulk_update", Payload is a JSON array of [name, data] pairs rather
// than a single event's data — see Tracker.Apply, which knows how to
// unwrap it. Every other event name's Payload is that event's own data
// object directly (matching the array's second element in the wire form
// `42["event_name", {...}]`).
type Event struct {
	Name    string
	Payload json.RawMessage
}

// ReadEvent blocks for the next Socket.IO event, transparently handling
// Engine.IO PING/PONG keepalive (the reporter's advertised pingInterval is
// an aggressive 5s — a caller that doesn't drain events promptly risks the
// server timing the connection out, since PONGs can only be sent from
// inside this read loop).
func (c *Client) ReadEvent() (Event, error) {
	for {
		op, payload, err := c.ws.ReadFrame()
		if err != nil {
			return Event{}, err
		}
		if op == opClose {
			return Event{}, fmt.Errorf("freedvreporter: connection closed by server")
		}
		if op != opText || len(payload) == 0 {
			continue
		}

		switch payload[0] {
		case '2': // Engine.IO PING -> reply PONG, keep waiting for a real event
			if werr := c.ws.WriteText("3"); werr != nil {
				return Event{}, fmt.Errorf("freedvreporter: send pong: %w", werr)
			}
			continue
		case '3': // Engine.IO PONG (shouldn't normally arrive; this client never pings) - ignore
			continue
		}

		if len(payload) < 2 || string(payload[:2]) != "42" {
			continue // some other Engine.IO/Socket.IO packet type - ignore
		}

		var arr []json.RawMessage
		if err := json.Unmarshal(payload[2:], &arr); err != nil || len(arr) == 0 {
			continue // malformed event frame - skip rather than fail the whole session
		}

		var name string
		if err := json.Unmarshal(arr[0], &name); err != nil {
			continue
		}

		var data json.RawMessage
		if len(arr) > 1 {
			data = arr[1]
		} else {
			data = json.RawMessage("null")
		}

		return Event{Name: name, Payload: data}, nil
	}
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
