# thetis-ai-skill

A Go CLI (`thetisctl`) that gives an AI agent remote control of a Thetis SDR
station, and a Python voice-operator subsystem (`talk/`) that hears and
answers callers over the radio on top of it.

## Language

**Dry run**:
The default behavior of every TX-capable command: it prints exactly what it would send and keys nothing.
_Avoid_: Simulation, preview

**TX-capable command**:
A `thetisctl` command or `talk` mode that can key a real transmitter into a real antenna if run for real.
_Avoid_: Transmit command, TX command

**Confirm phrase**:
The exact literal string (`internal/safety.ConfirmPhrase`) that must be passed to turn a dry run into a real transmission. Deliberately not a bare boolean, so nothing accidental can trigger it.
_Avoid_: Confirm flag, TX flag

**Control operator**:
The licensed human legally responsible for everything the station transmits, including what `talk` sends while armed. Never an agent, regardless of what was said earlier in a conversation.
_Avoid_: User, operator (alone)

### `talk` (voice operator)

**Utterance**:
One continuous stretch of received speech, as endpointed by `talk`'s voice-activity detector — bounded by silence before and after, or a forced cutoff at the maximum length.
_Avoid_: Speech segment, clip

**Turn**:
One cycle of the listening loop: an utterance is transcribed, checked for a Wake, and — if addressed — answered.
_Avoid_: Exchange, interaction

**Wake**:
A trigger that addresses the station: its phonetic callsign (fuzzy-matched) or a configured wake name. Only an utterance containing a Wake gets a reply; everything else is logged silently.
_Avoid_: Trigger word, hotword, activation phrase

**QSO**:
The conversational context spanning consecutive addressed Turns — opened at the first Wake, closed by a scripted sign-off, an idle timeout, or the QSO time cap.
_Avoid_: Conversation, contact, session

**Rehearsal mode**:
`talk`'s default posture: the full pipeline runs and replies play on local speakers, but the radio is never keyed.
_Avoid_: Simulation mode, test mode

**Armed session**:
The window between a human running `talk --armed` and it disarming — the only state in which `talk` actually transmits.
_Avoid_: Live mode, active session

**Disarm**:
`talk` giving up transmit capability mid-session — via keypress/Ctrl-C, session expiry, a failed pre-transmission check, or an unconfirmed unkey — without ending the process; it keeps listening.
_Avoid_: Abort, shutdown
