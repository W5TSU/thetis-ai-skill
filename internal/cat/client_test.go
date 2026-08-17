package cat

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

// newTestClient wires a Client to an in-memory net.Pipe() fake CAT server
// that replies to any request found in replies (request text without the
// trailing ';') and silently drops anything else, mirroring how Thetis
// treats fire-and-forget set commands.
func newTestClient(t *testing.T, replies map[string]string) (*Client, func()) {
	t.Helper()
	clientConn, serverConn := net.Pipe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r := bufio.NewReader(serverConn)
		for {
			line, err := r.ReadString(';')
			if err != nil {
				return
			}
			req := strings.TrimSuffix(line, ";")
			if reply, ok := replies[req]; ok {
				if _, err := serverConn.Write([]byte(reply + ";")); err != nil {
					return
				}
			}
		}
	}()

	c := newClient(clientConn, 2*time.Second)
	cleanup := func() {
		c.Close()
		serverConn.Close()
		<-done
	}
	return c, cleanup
}

func TestFreqRoundTrip(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"FA": "FA00014074000"})
	defer cleanup()

	if err := c.SetVFOFreqHz("A", 14074000); err != nil {
		t.Fatalf("SetVFOFreqHz: %v", err)
	}
	hz, err := c.GetVFOFreqHz("A")
	if err != nil {
		t.Fatalf("GetVFOFreqHz: %v", err)
	}
	if hz != 14074000 {
		t.Errorf("GetVFOFreqHz = %d, want 14074000", hz)
	}
}

func TestModeRoundTrip(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"MD": "MD2"})
	defer cleanup()

	if err := c.SetMode("USB"); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	mode, err := c.GetMode()
	if err != nil {
		t.Fatalf("GetMode: %v", err)
	}
	if mode != "USB" {
		t.Errorf("GetMode = %q, want USB", mode)
	}
}

func TestSplitRoundTrip(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZSP": "ZZSP1"})
	defer cleanup()

	if err := c.SetSplit(true); err != nil {
		t.Fatalf("SetSplit: %v", err)
	}
	on, err := c.GetSplit()
	if err != nil {
		t.Fatalf("GetSplit: %v", err)
	}
	if !on {
		t.Errorf("GetSplit = false, want true")
	}
}

func TestPowerRoundTrip(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZPS": "ZZPS1"})
	defer cleanup()

	if err := c.SetPowerOn(true); err != nil {
		t.Fatalf("SetPowerOn: %v", err)
	}
	on, err := c.GetPowerOn()
	if err != nil {
		t.Fatalf("GetPowerOn: %v", err)
	}
	if !on {
		t.Errorf("GetPowerOn = false, want true")
	}
}

func TestQuickPlayRoundTrip(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZQA": "ZZQA1"})
	defer cleanup()

	if err := c.SetQuickPlay(true); err != nil {
		t.Fatalf("SetQuickPlay: %v", err)
	}
	on, err := c.GetQuickPlay()
	if err != nil {
		t.Fatalf("GetQuickPlay: %v", err)
	}
	if !on {
		t.Errorf("GetQuickPlay = false, want true")
	}
}

func TestQuickRecRoundTrip(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZQB": "ZZQB1"})
	defer cleanup()

	if err := c.SetQuickRec(true); err != nil {
		t.Fatalf("SetQuickRec: %v", err)
	}
	on, err := c.GetQuickRec()
	if err != nil {
		t.Fatalf("GetQuickRec: %v", err)
	}
	if !on {
		t.Errorf("GetQuickRec = false, want true")
	}
}

func TestFreeDVDecodeRoundTrip(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZDV": "ZZDV1"})
	defer cleanup()

	if err := c.SetFreeDVDecode(true); err != nil {
		t.Fatalf("SetFreeDVDecode: %v", err)
	}
	on, err := c.GetFreeDVDecode()
	if err != nil {
		t.Fatalf("GetFreeDVDecode: %v", err)
	}
	if !on {
		t.Errorf("GetFreeDVDecode = false, want true")
	}
}

func TestFreeDVStatusSynced(t *testing.T) {
	// "1+153" = synced, 15.3dB SNR.
	c, cleanup := newTestClient(t, map[string]string{"ZZDS": "ZZDS1+153"})
	defer cleanup()

	st, err := c.GetFreeDVStatus()
	if err != nil {
		t.Fatalf("GetFreeDVStatus: %v", err)
	}
	if !st.Sync {
		t.Errorf("Sync = false, want true")
	}
	if st.SNRdB != 15.3 {
		t.Errorf("SNRdB = %v, want 15.3", st.SNRdB)
	}
}

func TestFreeDVStatusNotSynced(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZDS": "ZZDS0+000"})
	defer cleanup()

	st, err := c.GetFreeDVStatus()
	if err != nil {
		t.Fatalf("GetFreeDVStatus: %v", err)
	}
	if st.Sync {
		t.Errorf("Sync = true, want false")
	}
	if st.SNRdB != 0 {
		t.Errorf("SNRdB = %v, want 0", st.SNRdB)
	}
}

func TestFreeDVStatusNegativeSNR(t *testing.T) {
	// "1-025" = synced, -2.5dB SNR.
	c, cleanup := newTestClient(t, map[string]string{"ZZDS": "ZZDS1-025"})
	defer cleanup()

	st, err := c.GetFreeDVStatus()
	if err != nil {
		t.Fatalf("GetFreeDVStatus: %v", err)
	}
	if !st.Sync {
		t.Errorf("Sync = false, want true")
	}
	if st.SNRdB != -2.5 {
		t.Errorf("SNRdB = %v, want -2.5", st.SNRdB)
	}
}

func TestFreeDVStatusMalformedReply(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZDS": "ZZDSxx"})
	defer cleanup()

	if _, err := c.GetFreeDVStatus(); err == nil {
		t.Fatal("GetFreeDVStatus with malformed reply: want error, got nil")
	}
}

func TestQueryUnexpectedReplyIsError(t *testing.T) {
	c, cleanup := newTestClient(t, map[string]string{"ZZ": "XX1"})
	defer cleanup()

	if _, err := c.Query("ZZ"); err == nil {
		t.Fatal("Query with mismatched reply prefix: want error, got nil")
	}
}

// TestQuerySkipsWelcomeBanner reproduces a real Thetis instance sending its
// unsolicited "#Thetis TCP/IP Cat - <version>#;" banner (TCPIPcatServer.cs:98)
// immediately on connect, before any reply to the first query. Uses a real
// loopback TCP listener rather than net.Pipe: net.Pipe's writes block until
// matched by a read on the other end, which would deadlock this scenario
// (server wants to write the banner before reading anything; client wants to
// write its query before reading anything) — a real socket's kernel buffer
// absorbs the banner write immediately, exactly as happens against Thetis.
func TestQuerySkipsWelcomeBanner(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("#Thetis TCP/IP Cat - Thetis v2.10.3.19 x64#;"))
		r := bufio.NewReader(conn)
		line, err := r.ReadString(';')
		if err != nil {
			return
		}
		if strings.TrimSuffix(line, ";") == "ID" {
			conn.Write([]byte("ID019;"))
		}
	}()

	c, err := Dial(ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	id, err := c.Query("ID")
	if err != nil {
		t.Fatalf("Query(\"ID\") with a preceding welcome banner: %v", err)
	}
	if id != "019" {
		t.Errorf("Query(\"ID\") = %q, want \"019\"", id)
	}
}

func TestSetVFOFreqHzOutOfRange(t *testing.T) {
	c, cleanup := newTestClient(t, nil)
	defer cleanup()

	if err := c.SetVFOFreqHz("A", 999999999999); err == nil {
		t.Fatal("SetVFOFreqHz with 12-digit Hz: want error, got nil")
	}
}

func TestParseIF(t *testing.T) {
	// Field layout per CATCommands.cs IF() (lines 378-401):
	// freq(11) step(4) incr(6) rit(1) xit(1) dummy(3) tx(1) mode(1) dummy(2) split(1) dummy(4)
	s := "00014074000" + "0000" + "+00100" + "1" + "0" + "000" + "1" + "2" + "00" + "1" + "0000"
	if len(s) != 35 {
		t.Fatalf("test fixture length = %d, want 35", len(s))
	}

	st, err := parseIF(s)
	if err != nil {
		t.Fatalf("parseIF: %v", err)
	}
	if st.FreqHz != 14074000 {
		t.Errorf("FreqHz = %d, want 14074000", st.FreqHz)
	}
	if st.RITXITHz != 100 {
		t.Errorf("RITXITHz = %d, want 100", st.RITXITHz)
	}
	if !st.RIT {
		t.Errorf("RIT = false, want true")
	}
	if st.XIT {
		t.Errorf("XIT = true, want false")
	}
	if !st.TXActive {
		t.Errorf("TXActive = false, want true")
	}
	if st.Mode != "USB" {
		t.Errorf("Mode = %q, want USB", st.Mode)
	}
	if !st.Split {
		t.Errorf("Split = false, want true")
	}
}

func TestPTTWireCommands(t *testing.T) {
	var got []string
	clientConn, serverConn := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r := bufio.NewReader(serverConn)
		for {
			line, err := r.ReadString(';')
			if err != nil {
				return
			}
			got = append(got, strings.TrimSuffix(line, ";"))
		}
	}()
	c := newClient(clientConn, 2*time.Second)

	if err := c.SetPTT(true); err != nil {
		t.Fatalf("SetPTT(true): %v", err)
	}
	if err := c.SetPTT(false); err != nil {
		t.Fatalf("SetPTT(false): %v", err)
	}
	c.Close()
	serverConn.Close()
	<-done

	want := []string{"TX", "RX"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("wire commands = %v, want %v", got, want)
	}
}
