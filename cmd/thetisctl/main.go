// Command thetisctl remotely controls a running Thetis SDR instance over its
// CAT-over-TCP and TCI-over-WebSocket network protocols. See
// .claude/skills/thetis-control/SKILL.md for full usage and the mandatory
// TX safety protocol.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "cat":
		err = runCAT(os.Args[2:])
	case "tci":
		err = runTCI(os.Args[2:])
	case "freedv-reporter":
		err = runFreeDVReporter(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "thetisctl: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "thetisctl:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `thetisctl controls a running Thetis SDR instance over the network.
Thetis has no authentication on either protocol — only point this at a
trusted LAN, never the internet.

Usage:
  thetisctl cat --host <ip> [--port 13013] [--timeout 3s] <command> [args...]
  thetisctl tci --host <ip> [--port 50001] [--timeout 5s] <command> [args...]
  thetisctl freedv-reporter watch [--min-freq 14000000] [--max-freq 14350000]
                                   [--tci <ip>] [--tci-port 50001] [--mode digu]

CAT commands (control-only, Tier 1):
  freq get A|B                    freq set A|B <hz>
  mode get                        mode set <USB|LSB|CW|CWL|FM|AM|DIGU|DIGL>
  rit on|off|get                  xit on|off|get
  split on|off|get
  agc get                         agc set <FIXED|LONG|SLOW|MEDIUM|FAST|CUSTOM>
  atten get                       atten set <0-31>
  preamp set <0-9>
  band get                        band set <160|80|60|40|30|20|17|15|12|10|6|2|GEN|WWV|V0-V13>
  power get                       power on|off  (starts/stops Thetis's radio engine, not mains power)
  quickplay get                   quickplay on|off  (Quick Play: inject Music\Thetis\quickrecord\SDRQuickAudio.wav as RX I/Q, bypassing the antenna)
  quickrec get                    quickrec on|off   (Quick Rec: record RX audio to that same fixed file)
  freedv get                      freedv on|off  (enable/disable FreeDV RX decode, RX1 only)
  freedv status                   (read FreeDV sync + SNR)
  radae get                       radae on|off   (enable/disable RADE V1 RX decode, RX1 only)
  radae status                    (read RADE sync + SNR)
  radae-sanity --confirm-tx=<phrase> [--hold 140s] [--poll 1s] [--freq <hz>] [--mode <name>] [--csv <path>]
                                   (scripted off-air sanity check: radae on, quickplay on, poll status, summarize)
  status
  ptt on --confirm-tx=<phrase> [--hold 3s]     ptt off

TCI commands (control + audio + transmit, Tier 2; rx: 0=RX1, 1=RX2):
  vfo <rx> <chan 0|1> <hz>
  modulation <rx> <lsb|usb|dsb|am|sam|fm|cw|cwl|cwu|digl|digu>
  split <rx> on|off               rit <rx> on|off        xit <rx> on|off
  rit-offset <rx> <hz>            xit-offset <rx> <hz>
  filter <rx> <lowHz> <highHz>
  atten <rx> <0..N dB>            preamp <rx> <db<=0>
  agc <rx> <off|fixed|long|slow|medium|fast|custom>
  agc-gain <rx> <-20..120>
  drive <rx> <0-100>
  power on|off                    (starts/stops Thetis's radio engine, not mains power; waits for confirmation)
  tune <rx> on|off --confirm-tx=<phrase> [--hold 3s]  (bare carrier — always off within 5s total, capped regardless of --hold)
  ptt <rx> on|off --confirm-tx=<phrase> [--hold 3s] [--audio]
  rx-audio capture <rx> --duration 10s --out capture.wav [--sample-type float32]
  rx-audio stream <rx> --duration 10s        (raw float32 LE PCM to stdout)
  freedv-scan [--dwell 6s] [--out-dir <dir>]  (RX-only: tunes RX1 through FreeDV calling frequencies, records + peak/RMS per band)
  tx-audio send <rx> --file tone.wav --confirm-tx=<phrase> [--max-duration 10s] [--sample-type int16]
  cw send <rx> "<text>" --confirm-tx=<phrase> [--speed 20] [--max-duration 90s]
  query <cmd> [args...]           (raw passthrough for anything not above)

Every TX-capable command (ptt on, tune on, tx-audio send) defaults to a dry
run: it prints what it would send and does nothing TX-capable. Real keying
requires --confirm-tx equal to the exact phrase thetisctl prints in dry-run
output. Read the safety protocol in SKILL.md before ever passing that flag.

freedv-reporter watch: watches FreeDV Reporter's (qso.freedv.org) live feed
for stations starting to transmit within a frequency range (default 20m,
14.000-14.350 MHz). Prints an alert for each; with --tci, also retunes RX1
there automatically (not TX-capable — this only changes what Thetis is
listening to). Runs until Ctrl-C.
`)
}
