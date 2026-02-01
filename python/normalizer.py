"""Text normalization for ditong.

Handles multi-language character normalization to ASCII.
Supports Turkish, German, French, Spanish, and other Latin-script languages.
"""

import re
import unicodedata
from typing import Optional

# Comprehensive character mapping for multiple languages
CHAR_MAP: dict[str, str] = {
    # Turkish
    "ç": "c", "Ç": "c",
    "ş": "s", "Ş": "s",
    "ğ": "g", "Ğ": "g",
    "ı": "i", "İ": "i",
    # German
    "ä": "a", "Ä": "a",
    "ö": "o", "Ö": "o",
    "ü": "u", "Ü": "u",
    "ß": "ss",
    # French
    "à": "a", "â": "a", "æ": "ae",
    "é": "e", "è": "e", "ê": "e", "ë": "e",
    "î": "i", "ï": "i",
    "ô": "o", "œ": "oe",
    "ù": "u", "û": "u",
    "ÿ": "y",
    "ç": "c",
    # Spanish
    "á": "a", "í": "i", "ó": "o", "ú": "u",
    "ñ": "n", "Ñ": "n",
    # Portuguese
    "ã": "a", "õ": "o",
    # Polish
    "ą": "a", "ć": "c", "ę": "e", "ł": "l",
    "ń": "n", "ś": "s", "ź": "z", "ż": "z",
    # Czech/Slovak
    "č": "c", "ď": "d", "ě": "e", "ň": "n",
    "ř": "r", "š": "s", "ť": "t", "ů": "u", "ž": "z",
    # Nordic
    "å": "a", "Å": "a",
    "ø": "o", "Ø": "o",
    # Romanian
    "ă": "a", "ț": "t", "ș": "s",
}

ALPHA_PATTERN = re.compile(r"^[a-z]+$")


def normalize_char(char: str) -> str:
    """Normalize a single character to ASCII equivalent.

    Args:
        char: Single character.

    Returns:
        ASCII equivalent (may be multiple chars for ligatures like ß→ss).
    """
    # Check direct mapping first
    if char in CHAR_MAP:
        return CHAR_MAP[char]

    # Try lowercase version
    lower = char.lower()
    if lower in CHAR_MAP:
        return CHAR_MAP[lower]

    # Fall back to Unicode decomposition
    # This handles accented characters not in our map
    normalized = unicodedata.normalize("NFD", char)
    ascii_chars = []
    for c in normalized:
        if unicodedata.category(c) != "Mn":  # Not a combining mark
            if c.isascii():
                ascii_chars.append(c.lower())
    return "".join(ascii_chars) if ascii_chars else char.lower()


def normalize_word(word: str) -> str:
    """Normalize a word to lowercase ASCII.

    Applies character mapping and removes non-alphabetic characters.

    Args:
        word: Word to normalize.

    Returns:
        Normalized word (lowercase ASCII only).
    """
    result = []
    for char in word:
        result.append(normalize_char(char))
    return "".join(result)


def is_valid_identifier(word: str) -> bool:
    """Check if normalized word is valid (ASCII a-z only).

    Args:
        word: Normalized word to check.

    Returns:
        True if word contains only a-z characters.
    """
    return bool(ALPHA_PATTERN.match(word))


def normalize_and_validate(word: str) -> Optional[str]:
    """Normalize word and return if valid, else None.

    Args:
        word: Raw word.

    Returns:
        Normalized word or None if invalid.
    """
    normalized = normalize_word(word)
    if is_valid_identifier(normalized):
        return normalized
    return None
