"""Decide whether a transcript addresses this station.

Fuzzy on purpose: Whisper transcribing noisy SSB drops and mangles words, so
the phonetic callsign requires only 3 of its words, in order, inside a
7-token window, each word matched by difflib ratio rather than equality.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from difflib import SequenceMatcher

from talk.config import StationConfig

WINDOW_TOKENS = 7
MIN_PHONETIC_WORDS = 3
WORD_RATIO = 0.75
WAKE_RATIO = 0.85


@dataclass(frozen=True)
class TriggerMatch:
    triggered: bool
    kind: str | None = None  # "callsign" | "wake"
    score: float = 0.0
    matched_words: tuple[str, ...] = ()


def _tokens(text: str) -> list[str]:
    return re.findall(r"[a-z0-9']+", text.lower())


def _similar(a: str, b: str) -> float:
    return SequenceMatcher(None, a, b).ratio()


def _callsign_hits(window: list[str], phonetics: tuple[str, ...]) -> list[str]:
    """Greedy in-order matching of phonetic words against a token window."""
    hits = []
    i = 0
    for word in phonetics:
        for j in range(i, len(window)):
            if _similar(window[j], word) >= WORD_RATIO:
                hits.append(word)
                i = j + 1
                break
    return hits


def match(text: str, station: StationConfig) -> TriggerMatch:
    tokens = _tokens(text)
    if not tokens:
        return TriggerMatch(False)

    for token in tokens:
        for name in station.wake_names:
            score = _similar(token, name.lower())
            if score >= WAKE_RATIO:
                return TriggerMatch(True, "wake", score, (name,))

    phonetics = tuple(w.lower() for w in station.phonetic_words)
    best: list[str] = []
    for start in range(len(tokens)):
        window = tokens[start : start + WINDOW_TOKENS]
        hits = _callsign_hits(window, phonetics)
        if len(hits) > len(best):
            best = hits
    if len(best) >= MIN_PHONETIC_WORDS:
        return TriggerMatch(True, "callsign", len(best) / len(phonetics), tuple(best))
    return TriggerMatch(False, score=len(best) / len(phonetics) if phonetics else 0.0)
