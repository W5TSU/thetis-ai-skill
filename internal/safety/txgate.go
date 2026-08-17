// Package safety gates every code path in thetisctl that can key the
// transmitter. Neither of Thetis's network control protocols (CAT-over-TCP,
// TCI-over-WebSocket) has authentication, and TCI TX audio genuinely
// modulates and transmits RF (see Console/cmaster.cs TCITxThreadProc) — so
// anything TX-capable defaults to a dry run, and real keying requires an
// explicit, hard-to-fat-finger confirmation.
package safety

// ConfirmPhrase is the exact value --confirm-tx must equal to key the
// transmitter. A bare boolean flag is intentionally rejected so that no
// other tool's "--confirm-tx" convention can accidentally authorize TX here.
const ConfirmPhrase = "I-UNDERSTAND-THIS-KEYS-THE-RADIO"

// Decision is the outcome of a TX confirmation check.
type Decision struct {
	Proceed bool
	DryRun  bool
}

// Check validates a --confirm-tx flag value against ConfirmPhrase.
//
// This is thetisctl's only realtime gate. The harder requirement — that
// nothing ever passes this flag without a human explicitly authorizing this
// specific transmission (frequency, mode, content, duration) in the current
// conversation — is procedural, enforced by
// .claude/skills/thetis-control/SKILL.md's safety protocol, not by this
// function; a CLI flag cannot verify who typed it or why.
//
// An earlier version of this gate also required an interactive TTY "yes"
// prompt on top of the phrase match. That was removed: it assumed a real
// terminal is attached to read the prompt from, but thetisctl's primary use
// case is an AI agent invoking it non-interactively on the operator's behalf
// after collecting consent through its own conversation — stdin there is
// typically closed or redirected (e.g. /dev/null), which a naive
// os.ModeCharDevice check misidentifies as a TTY, so the prompt blocked on
// EOF instead of protecting anything.
func Check(confirmFlag string) Decision {
	if confirmFlag != ConfirmPhrase {
		return Decision{Proceed: false, DryRun: true}
	}
	return Decision{Proceed: true, DryRun: false}
}
