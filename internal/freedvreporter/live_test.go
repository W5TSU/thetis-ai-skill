//go:build live

// Live test against the real FreeDV Reporter service (qso.freedv.org).
// Excluded from normal `go test ./...` and CI by the "live" build tag, same
// convention as internal/tci and internal/cat's live tests — run explicitly:
//
//	go test -tags=live ./internal/freedvreporter/... -v
//
// This only reads the public activity feed; it never touches a Thetis
// instance or any radio hardware.
package freedvreporter

import (
	"testing"
	"time"
)

func TestLiveConnectAndTrack(t *testing.T) {
	c, err := Dial(10 * time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	tracker := NewTracker()
	deadline := time.Now().Add(15 * time.Second)
	eventCount := 0
	for time.Now().Before(deadline) {
		ev, err := c.ReadEvent()
		if err != nil {
			t.Fatalf("ReadEvent: %v", err)
		}
		eventCount++
		if _, err := tracker.Apply(ev); err != nil {
			t.Errorf("Apply(%s): %v", ev.Name, err)
		}
	}

	if eventCount == 0 {
		t.Fatal("received zero events in 15s - reporter unreachable or protocol broke")
	}

	stations := tracker.Stations()
	t.Logf("received %d events, tracked %d stations", eventCount, len(stations))

	haveCallsign := false
	for _, s := range stations {
		if s.Callsign != "" {
			haveCallsign = true
			t.Logf("  %-10s freq=%dHz mode=%q tx=%v rxonly=%v", s.Callsign, s.FreqHz, s.Mode, s.Transmitting, s.RXOnly)
		}
	}
	if !haveCallsign {
		t.Error("tracked stations but none had a callsign - new_connection parsing likely broken")
	}
}
