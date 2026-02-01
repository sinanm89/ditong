"""Phonetics module for ditong.

Provides pluggable transcription backends and phonetic similarity search.

Usage:
    from ditong.phonetics import transcribe, get_transcriber

    # Use default (epitran)
    ipa = transcribe("hello", "en")

    # Use specific backend
    transcriber = get_transcriber("espeak")
    ipa = transcriber.transcribe("hello", "en")

    # Batch transcribe
    from ditong.phonetics import batch_transcribe
    results = batch_transcribe(["hello", "world"], "en")
"""

from .transcribe import (
    Transcriber,
    transcribe,
    batch_transcribe,
    get_transcriber,
    register_transcriber,
    list_transcribers,
    get_default_transcriber,
    set_default_transcriber,
)

__all__ = [
    "Transcriber",
    "transcribe",
    "batch_transcribe",
    "get_transcriber",
    "register_transcriber",
    "list_transcribers",
    "get_default_transcriber",
    "set_default_transcriber",
]
