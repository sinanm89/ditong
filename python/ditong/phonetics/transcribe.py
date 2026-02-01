"""Pluggable IPA transcription backends.

Supports multiple transcription engines with a unified interface.
Each backend is a simple function or class that can be imported and used directly.

Backends:
    - epitran: Pure Python, good multilingual support
    - espeak: Uses espeak-ng (requires system install)
    - phonemizer: Wraps espeak/festival (requires system install)
    - g2p_en: Neural G2P for English only
    - gruut: Multilingual neural (larger models)

Usage:
    # Simple function call
    from ditong.phonetics import transcribe
    ipa = transcribe("hello", "en")

    # Get specific backend
    from ditong.phonetics import get_transcriber
    espeak = get_transcriber("espeak")
    ipa = espeak.transcribe("hello", "en")

    # Register custom backend
    from ditong.phonetics import register_transcriber
    register_transcriber("custom", MyTranscriberClass)
"""

from abc import ABC, abstractmethod
from typing import Optional, Callable

# Registry of available transcribers
_TRANSCRIBERS: dict[str, type["Transcriber"]] = {}
_DEFAULT_TRANSCRIBER: str = "epitran"


class Transcriber(ABC):
    """Base class for IPA transcription backends."""

    name: str = "base"
    supported_languages: list[str] = []

    @abstractmethod
    def transcribe(self, word: str, language: str) -> Optional[str]:
        """Transcribe a single word to IPA.

        Args:
            word: Word to transcribe.
            language: Language code (e.g., "en", "tr").

        Returns:
            IPA transcription or None if failed.
        """
        pass

    def batch_transcribe(
        self,
        words: list[str],
        language: str,
        skip_errors: bool = True
    ) -> dict[str, Optional[str]]:
        """Transcribe multiple words.

        Args:
            words: List of words.
            language: Language code.
            skip_errors: If True, return None for failures.

        Returns:
            Dict mapping words to IPA.
        """
        results = {}
        for word in words:
            try:
                results[word] = self.transcribe(word, language)
            except Exception:
                if skip_errors:
                    results[word] = None
                else:
                    raise
        return results

    def supports_language(self, language: str) -> bool:
        """Check if language is supported."""
        return language in self.supported_languages


# =============================================================================
# Epitran Backend
# =============================================================================

class EpitranTranscriber(Transcriber):
    """Epitran-based transcriber.

    Pure Python, good multilingual support.
    Install: pip install epitran
    """

    name = "epitran"
    supported_languages = ["en", "tr", "de", "fr", "es", "it", "nl", "pl", "ru"]

    # Epitran language codes
    _LANG_MAP = {
        "en": "eng-Latn",
        "tr": "tur-Latn",
        "de": "deu-Latn",
        "fr": "fra-Latn",
        "es": "spa-Latn",
        "it": "ita-Latn",
        "nl": "nld-Latn",
        "pl": "pol-Latn",
        "ru": "rus-Cyrl",
    }

    def __init__(self):
        self._instances: dict[str, "epitran.Epitran"] = {}

    def _get_instance(self, language: str):
        if language not in self._instances:
            try:
                import epitran
            except ImportError as e:
                raise ImportError(
                    "epitran required. Install: pip install epitran"
                ) from e

            if language not in self._LANG_MAP:
                raise ValueError(f"Unsupported language: {language}")

            self._instances[language] = epitran.Epitran(self._LANG_MAP[language])

        return self._instances[language]

    def transcribe(self, word: str, language: str) -> Optional[str]:
        try:
            epi = self._get_instance(language)
            return epi.transliterate(word)
        except Exception:
            return None


# =============================================================================
# Espeak Backend
# =============================================================================

class EspeakTranscriber(Transcriber):
    """Espeak-ng based transcriber.

    Requires espeak-ng system install.
    Windows: Download from https://github.com/espeak-ng/espeak-ng/releases
    Linux: apt install espeak-ng
    macOS: brew install espeak-ng
    """

    name = "espeak"
    supported_languages = ["en", "tr", "de", "fr", "es", "it", "nl", "pl", "ru", "pt"]

    _LANG_MAP = {
        "en": "en-us",
        "tr": "tr",
        "de": "de",
        "fr": "fr",
        "es": "es",
        "it": "it",
        "nl": "nl",
        "pl": "pl",
        "ru": "ru",
        "pt": "pt",
    }

    def transcribe(self, word: str, language: str) -> Optional[str]:
        try:
            import subprocess

            if language not in self._LANG_MAP:
                return None

            lang_code = self._LANG_MAP[language]
            result = subprocess.run(
                ["espeak-ng", "-v", lang_code, "-q", "--ipa", word],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode == 0:
                return result.stdout.strip()
            return None
        except Exception:
            return None


# =============================================================================
# Phonemizer Backend
# =============================================================================

class PhonemizerTranscriber(Transcriber):
    """Phonemizer-based transcriber.

    Wraps espeak/festival backends with cleaner interface.
    Install: pip install phonemizer
    Requires: espeak-ng system install
    """

    name = "phonemizer"
    supported_languages = ["en", "tr", "de", "fr", "es", "it"]

    _LANG_MAP = {
        "en": "en-us",
        "tr": "tr",
        "de": "de",
        "fr": "fr-fr",
        "es": "es",
        "it": "it",
    }

    def transcribe(self, word: str, language: str) -> Optional[str]:
        try:
            from phonemizer import phonemize
            from phonemizer.backend import EspeakBackend

            if language not in self._LANG_MAP:
                return None

            result = phonemize(
                word,
                language=self._LANG_MAP[language],
                backend="espeak",
                strip=True,
            )
            return result if result else None
        except Exception:
            return None


# =============================================================================
# G2P English Backend
# =============================================================================

class G2PEnglishTranscriber(Transcriber):
    """G2P neural transcriber for English.

    High quality but English only.
    Install: pip install g2p-en
    """

    name = "g2p_en"
    supported_languages = ["en"]

    def __init__(self):
        self._g2p = None

    def _get_instance(self):
        if self._g2p is None:
            try:
                from g2p_en import G2p
            except ImportError as e:
                raise ImportError(
                    "g2p-en required. Install: pip install g2p-en"
                ) from e
            self._g2p = G2p()
        return self._g2p

    def transcribe(self, word: str, language: str) -> Optional[str]:
        if language != "en":
            return None
        try:
            g2p = self._get_instance()
            phonemes = g2p(word)
            return "".join(phonemes)
        except Exception:
            return None


# =============================================================================
# Gruut Backend
# =============================================================================

class GruutTranscriber(Transcriber):
    """Gruut multilingual transcriber.

    Offline neural models, good quality.
    Install: pip install gruut
    """

    name = "gruut"
    supported_languages = ["en", "de", "fr", "es", "it", "nl", "ru"]

    _LANG_MAP = {
        "en": "en-us",
        "de": "de-de",
        "fr": "fr-fr",
        "es": "es-es",
        "it": "it-it",
        "nl": "nl",
        "ru": "ru-ru",
    }

    def transcribe(self, word: str, language: str) -> Optional[str]:
        try:
            from gruut import sentences

            if language not in self._LANG_MAP:
                return None

            for sent in sentences(word, lang=self._LANG_MAP[language]):
                phonemes = []
                for word_obj in sent:
                    if word_obj.phonemes:
                        phonemes.extend(word_obj.phonemes)
                return " ".join(phonemes) if phonemes else None
            return None
        except Exception:
            return None


# =============================================================================
# Registry Functions
# =============================================================================

def _init_registry():
    """Initialize the transcriber registry with built-in backends."""
    global _TRANSCRIBERS
    _TRANSCRIBERS = {
        "epitran": EpitranTranscriber,
        "espeak": EspeakTranscriber,
        "phonemizer": PhonemizerTranscriber,
        "g2p_en": G2PEnglishTranscriber,
        "gruut": GruutTranscriber,
    }


_init_registry()

# Cached instances
_INSTANCES: dict[str, Transcriber] = {}


def get_transcriber(name: str) -> Transcriber:
    """Get a transcriber instance by name.

    Args:
        name: Transcriber name.

    Returns:
        Transcriber instance (cached).
    """
    if name not in _TRANSCRIBERS:
        raise ValueError(
            f"Unknown transcriber: {name}. "
            f"Available: {list(_TRANSCRIBERS.keys())}"
        )

    if name not in _INSTANCES:
        _INSTANCES[name] = _TRANSCRIBERS[name]()

    return _INSTANCES[name]


def register_transcriber(name: str, cls: type[Transcriber]) -> None:
    """Register a custom transcriber.

    Args:
        name: Name to register under.
        cls: Transcriber class.
    """
    _TRANSCRIBERS[name] = cls
    # Clear cached instance if exists
    if name in _INSTANCES:
        del _INSTANCES[name]


def list_transcribers() -> list[str]:
    """List available transcriber names."""
    return list(_TRANSCRIBERS.keys())


def get_default_transcriber() -> str:
    """Get the default transcriber name."""
    return _DEFAULT_TRANSCRIBER


def set_default_transcriber(name: str) -> None:
    """Set the default transcriber."""
    global _DEFAULT_TRANSCRIBER
    if name not in _TRANSCRIBERS:
        raise ValueError(f"Unknown transcriber: {name}")
    _DEFAULT_TRANSCRIBER = name


# =============================================================================
# Convenience Functions
# =============================================================================

def transcribe(
    word: str,
    language: str,
    backend: Optional[str] = None
) -> Optional[str]:
    """Transcribe a word to IPA using the default or specified backend.

    Args:
        word: Word to transcribe.
        language: Language code.
        backend: Backend name (uses default if None).

    Returns:
        IPA transcription or None.
    """
    backend_name = backend or _DEFAULT_TRANSCRIBER
    transcriber = get_transcriber(backend_name)
    return transcriber.transcribe(word, language)


def batch_transcribe(
    words: list[str],
    language: str,
    backend: Optional[str] = None,
    skip_errors: bool = True
) -> dict[str, Optional[str]]:
    """Batch transcribe words to IPA.

    Args:
        words: List of words.
        language: Language code.
        backend: Backend name (uses default if None).
        skip_errors: If True, return None for failures.

    Returns:
        Dict mapping words to IPA.
    """
    backend_name = backend or _DEFAULT_TRANSCRIBER
    transcriber = get_transcriber(backend_name)
    return transcriber.batch_transcribe(words, language, skip_errors=skip_errors)
