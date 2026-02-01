"""Tests for the normalizer module."""

import pytest
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

from normalizer import (
    normalize_char,
    normalize_word,
    is_valid_identifier,
    normalize_and_validate,
    CHAR_MAP,
)


class TestNormalizeChar:
    """Tests for normalize_char function."""

    def test_turkish_chars(self):
        """Test Turkish character normalization."""
        assert normalize_char("ç") == "c"
        assert normalize_char("Ç") == "c"
        assert normalize_char("ş") == "s"
        assert normalize_char("Ş") == "s"
        assert normalize_char("ğ") == "g"
        assert normalize_char("Ğ") == "g"
        assert normalize_char("ı") == "i"
        assert normalize_char("İ") == "i"
        assert normalize_char("ö") == "o"
        assert normalize_char("Ö") == "o"
        assert normalize_char("ü") == "u"
        assert normalize_char("Ü") == "u"

    def test_german_chars(self):
        """Test German character normalization."""
        assert normalize_char("ä") == "a"
        assert normalize_char("Ä") == "a"
        assert normalize_char("ö") == "o"
        assert normalize_char("ü") == "u"
        assert normalize_char("ß") == "ss"

    def test_french_chars(self):
        """Test French character normalization."""
        assert normalize_char("é") == "e"
        assert normalize_char("è") == "e"
        assert normalize_char("ê") == "e"
        assert normalize_char("à") == "a"
        assert normalize_char("â") == "a"

    def test_spanish_chars(self):
        """Test Spanish character normalization."""
        assert normalize_char("ñ") == "n"
        assert normalize_char("Ñ") == "n"
        assert normalize_char("á") == "a"
        assert normalize_char("é") == "e"

    def test_ascii_passthrough(self):
        """Test ASCII characters pass through unchanged."""
        assert normalize_char("a") == "a"
        assert normalize_char("z") == "z"
        assert normalize_char("A") == "a"
        assert normalize_char("Z") == "z"

    def test_numbers_passthrough(self):
        """Test numbers pass through."""
        assert normalize_char("0") == "0"
        assert normalize_char("9") == "9"


class TestNormalizeWord:
    """Tests for normalize_word function."""

    def test_simple_words(self):
        """Test simple ASCII words."""
        assert normalize_word("hello") == "hello"
        assert normalize_word("HELLO") == "hello"
        assert normalize_word("Hello") == "hello"

    def test_turkish_words(self):
        """Test Turkish words normalize correctly."""
        assert normalize_word("çare") == "care"
        assert normalize_word("şeker") == "seker"
        assert normalize_word("görmek") == "gormek"
        assert normalize_word("ışık") == "isik"

    def test_german_words(self):
        """Test German words normalize correctly."""
        assert normalize_word("größe") == "grosse"
        assert normalize_word("über") == "uber"
        assert normalize_word("Mädchen") == "madchen"

    def test_mixed_case(self):
        """Test mixed case words."""
        assert normalize_word("ÇARE") == "care"
        assert normalize_word("Größe") == "grosse"

    def test_care_ccare_equivalence(self):
        """Test that care (EN) and çare (TR) normalize to same value."""
        assert normalize_word("care") == normalize_word("çare")


class TestIsValidIdentifier:
    """Tests for is_valid_identifier function."""

    def test_valid_identifiers(self):
        """Test valid ASCII-only identifiers."""
        assert is_valid_identifier("hello") is True
        assert is_valid_identifier("world") is True
        assert is_valid_identifier("a") is True
        assert is_valid_identifier("abcdefghijklmnopqrstuvwxyz") is True

    def test_invalid_with_numbers(self):
        """Test identifiers with numbers are invalid."""
        assert is_valid_identifier("hello123") is False
        assert is_valid_identifier("123") is False
        assert is_valid_identifier("abc1") is False

    def test_invalid_with_special_chars(self):
        """Test identifiers with special chars are invalid."""
        assert is_valid_identifier("hello-world") is False
        assert is_valid_identifier("hello_world") is False
        assert is_valid_identifier("hello world") is False
        assert is_valid_identifier("hello!") is False

    def test_invalid_with_uppercase(self):
        """Test uppercase letters are invalid (must be lowercase)."""
        assert is_valid_identifier("Hello") is False
        assert is_valid_identifier("HELLO") is False

    def test_empty_string(self):
        """Test empty string is invalid."""
        assert is_valid_identifier("") is False


class TestNormalizeAndValidate:
    """Tests for normalize_and_validate function."""

    def test_valid_words(self):
        """Test valid words return normalized form."""
        assert normalize_and_validate("hello") == "hello"
        assert normalize_and_validate("HELLO") == "hello"
        assert normalize_and_validate("çare") == "care"

    def test_invalid_words(self):
        """Test invalid words return None."""
        assert normalize_and_validate("hello123") is None
        assert normalize_and_validate("hello-world") is None
        assert normalize_and_validate("") is None

    def test_special_chars_removed(self):
        """Test words with non-letter chars return None after normalization."""
        assert normalize_and_validate("hello!") is None
        assert normalize_and_validate("test@word") is None


class TestCharMap:
    """Tests for the CHAR_MAP dictionary."""

    def test_char_map_completeness(self):
        """Test that common special chars are in the map."""
        turkish = ["ç", "Ç", "ş", "Ş", "ğ", "Ğ", "ı", "İ", "ö", "Ö", "ü", "Ü"]
        for char in turkish:
            assert char in CHAR_MAP, f"Turkish char {char} not in CHAR_MAP"

    def test_char_map_values_are_ascii(self):
        """Test all mapped values are ASCII."""
        for char, mapped in CHAR_MAP.items():
            assert mapped.isascii(), f"{char} maps to non-ASCII: {mapped}"
            assert mapped.islower() or mapped == "ss", f"{char} maps to non-lower: {mapped}"
