package freedvreporter

import (
	"encoding/json"
	"sync"
	"time"
)

// Station is this package's local mirror of one reporter connection's state,
// built up from new_connection/freq_change/tx_report/remove_connection
// events. Frequency and transmit state arrive as separate events for the
// same station (keyed by SID) — a Station only becomes useful to act on once
// both have been seen at least once.
type Station struct {
	SID          string
	Callsign     string
	GridSquare   string
	Version      string
	RXOnly       bool
	FreqHz       int64 // 0 until a freq_change event has been seen for this SID
	Mode         string
	Transmitting bool
	LastUpdate   time.Time
	LastTx       time.Time
}

// TxStarted is emitted by Tracker.Apply when a tx_report event transitions a
// station from not-transmitting to transmitting — the actual "someone just
// keyed up" signal this package exists to surface. Station is a snapshot at
// the moment of the transition (in particular, FreqHz is whatever the last
// freq_change for this SID reported — it is not part of tx_report itself).
type TxStarted struct {
	Station Station
}

// Tracker maintains live station state from a stream of Events and reports
// transmit-start transitions. Safe for concurrent use.
type Tracker struct {
	mu       sync.Mutex
	stations map[string]*Station // keyed by SID
}

// NewTracker returns an empty Tracker.
func NewTracker() *Tracker {
	return &Tracker{stations: make(map[string]*Station)}
}

// Apply processes one Event (as returned by Client.ReadEvent), updating
// internal state, and returns every transmit-start transition it caused. A
// "bulk_update" event unwraps to zero or more inner events, each applied the
// same way — this is the only event name that can produce more than one
// TxStarted from a single Apply call.
func (t *Tracker) Apply(ev Event) ([]TxStarted, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ev.Name == "bulk_update" {
		var items []json.RawMessage
		if err := json.Unmarshal(ev.Payload, &items); err != nil {
			return nil, nil // malformed - not fatal, just nothing to apply
		}
		var out []TxStarted
		for _, item := range items {
			var pair []json.RawMessage
			if err := json.Unmarshal(item, &pair); err != nil || len(pair) < 2 {
				continue
			}
			var name string
			if err := json.Unmarshal(pair[0], &name); err != nil {
				continue
			}
			if s := t.applyOne(name, pair[1]); s != nil {
				out = append(out, *s)
			}
		}
		return out, nil
	}

	if s := t.applyOne(ev.Name, ev.Payload); s != nil {
		return []TxStarted{*s}, nil
	}
	return nil, nil
}

// applyOne handles a single, already-unwrapped (name, payload) pair. Caller
// must hold t.mu.
func (t *Tracker) applyOne(name string, payload json.RawMessage) *TxStarted {
	switch name {
	case "new_connection":
		var d struct {
			SID        string `json:"sid"`
			Callsign   string `json:"callsign"`
			GridSquare string `json:"grid_square"`
			Version    string `json:"version"`
			RXOnly     bool   `json:"rx_only"`
		}
		if json.Unmarshal(payload, &d) != nil || d.SID == "" {
			return nil
		}
		t.stations[d.SID] = &Station{
			SID:        d.SID,
			Callsign:   d.Callsign,
			GridSquare: d.GridSquare,
			Version:    d.Version,
			RXOnly:     d.RXOnly,
		}

	case "remove_connection":
		var d struct {
			SID string `json:"sid"`
		}
		if json.Unmarshal(payload, &d) == nil && d.SID != "" {
			delete(t.stations, d.SID)
		}

	case "freq_change":
		var d struct {
			SID  string `json:"sid"`
			Freq int64  `json:"freq"`
		}
		if json.Unmarshal(payload, &d) != nil || d.SID == "" {
			return nil
		}
		s := t.stations[d.SID]
		if s == nil {
			// freq_change can arrive before new_connection has been applied
			// (e.g. reordered within one bulk_update) - seed a bare entry
			// rather than dropping the frequency.
			s = &Station{SID: d.SID}
			t.stations[d.SID] = s
		}
		s.FreqHz = d.Freq
		s.LastUpdate = time.Now()

	case "tx_report":
		var d struct {
			SID          string `json:"sid"`
			Callsign     string `json:"callsign"`
			Mode         string `json:"mode"`
			Transmitting bool   `json:"transmitting"`
			LastTx       string `json:"last_tx"`
		}
		if json.Unmarshal(payload, &d) != nil || d.SID == "" {
			return nil
		}
		s := t.stations[d.SID]
		if s == nil {
			s = &Station{SID: d.SID}
			t.stations[d.SID] = s
		}
		if d.Callsign != "" {
			s.Callsign = d.Callsign
		}
		wasTransmitting := s.Transmitting
		s.Mode = d.Mode
		s.Transmitting = d.Transmitting
		if t, err := time.Parse(time.RFC3339Nano, d.LastTx); err == nil {
			s.LastTx = t
		}
		s.LastUpdate = time.Now()

		if d.Transmitting && !wasTransmitting {
			return &TxStarted{Station: *s}
		}

	// "rx_report" (someone reporting they heard another station, with SNR),
	// "chat_message", "chat_login", "chat_logout", "connection_successful"
	// are all seen on the wire but not needed for tune-to-active-TX
	// purposes - deliberately ignored rather than erroring on unknowns.
	default:
	}
	return nil
}

// Stations returns a snapshot of every currently-tracked station.
func (t *Tracker) Stations() []Station {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]Station, 0, len(t.stations))
	for _, s := range t.stations {
		out = append(out, *s)
	}
	return out
}
