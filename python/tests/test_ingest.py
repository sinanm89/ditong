"""Tests for the ingest module."""

import pytest
import tempfile
from pathlib import Path

from ditong.ingest.base import Ingestor, IngestResult
from ditong.ingest.hunspell import HunspellIngestor, HUNSPELL_URLS
from ditong.ingest.plain_text import PlainTextIngestor


class TestIngestResult:
    """Tests for IngestResult dataclass."""

    def test_repr(self):
        """Test IngestResult string representation."""
        result = IngestResult(
            words=[],
            source_path="/test.dic",
            dict_name="test",
            language="en",
            category="standard",
            total_raw=100,
            total_valid=80,
            total_duplicates=5,
        )
        repr_str = repr(result)
        assert "test" in repr_str
        assert "80/100" in repr_str


class TestHunspellIngestor:
    """Tests for HunspellIngestor."""

    def test_supported_languages(self):
        """Test that common languages have URLs."""
        assert "en" in HUNSPELL_URLS
        assert "tr" in HUNSPELL_URLS
        assert "de" in HUNSPELL_URLS
        assert "fr" in HUNSPELL_URLS

    def test_parse_simple_dic(self):
        """Test parsing a simple Hunspell .dic file."""
        content = """10
hello
world
testing/ABC
sample
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".dic", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = HunspellIngestor(
                language="en",
                cache_dir=filepath.parent,
            )
            words = list(ingestor.parse(filepath))

            assert len(words) == 4
            # Check word extraction
            word_forms = [w[0] for w in words]
            assert "hello" in word_forms
            assert "world" in word_forms
            assert "testing" in word_forms  # Stripped /ABC
            assert "sample" in word_forms
        finally:
            filepath.unlink()

    def test_parse_with_turkish_chars(self):
        """Test parsing Turkish characters."""
        content = """5
çare
şeker
merhaba
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".dic", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = HunspellIngestor(
                language="tr",
                cache_dir=filepath.parent,
            )
            words = list(ingestor.parse(filepath))

            word_forms = [w[0] for w in words]
            assert "çare" in word_forms
            assert "şeker" in word_forms
        finally:
            filepath.unlink()

    def test_ingest_filters_by_length(self):
        """Test that ingest filters words by length."""
        content = """10
a
ab
abc
abcd
abcde
abcdef
abcdefg
abcdefgh
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".dic", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = HunspellIngestor(
                language="en",
                cache_dir=filepath.parent,
                min_length=4,
                max_length=6,
            )
            result = ingestor.ingest(filepath)

            # Should only include 4, 5, 6 letter words
            lengths = [w.length for w in result.words]
            assert all(4 <= l <= 6 for l in lengths)
            assert 4 in lengths
            assert 5 in lengths
            assert 6 in lengths
        finally:
            filepath.unlink()

    def test_get_dict_name(self):
        """Test dictionary name generation."""
        ingestor = HunspellIngestor(language="en", cache_dir="/tmp")
        assert ingestor.get_dict_name(Path("/path/to/file.dic")) == "hunspell_en"


class TestPlainTextIngestor:
    """Tests for PlainTextIngestor."""

    def test_parse_simple_list(self):
        """Test parsing a simple word list."""
        content = """# This is a comment
hello
world
# Another comment
test
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".txt", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = PlainTextIngestor(language="en")
            words = list(ingestor.parse(filepath))

            word_forms = [w[0] for w in words]
            assert "hello" in word_forms
            assert "world" in word_forms
            assert "test" in word_forms
            # Comments should not be included
            assert len(words) == 3
        finally:
            filepath.unlink()

    def test_parse_with_inline_comments(self):
        """Test parsing with inline comments."""
        content = """hello # greeting
world # planet
test
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".txt", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = PlainTextIngestor(language="en")
            words = list(ingestor.parse(filepath))

            word_forms = [w[0] for w in words]
            assert "hello" in word_forms
            assert "world" in word_forms
        finally:
            filepath.unlink()

    def test_custom_comment_char(self):
        """Test custom comment character."""
        content = """; This is a comment
hello
world
; Another comment
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".txt", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = PlainTextIngestor(language="en", comment_char=";")
            words = list(ingestor.parse(filepath))

            assert len(words) == 2
        finally:
            filepath.unlink()

    def test_ingest_with_category(self):
        """Test ingest with custom category."""
        content = """badword
anotherbad
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".txt", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = PlainTextIngestor(
                language="en",
                category="curseword",
                min_length=3,
                max_length=20,
            )
            result = ingestor.ingest(filepath)

            for word in result.words:
                assert "curseword" in word.categories
        finally:
            filepath.unlink()


class TestIngestIntegration:
    """Integration tests for ingest module."""

    def test_normalization_in_ingest(self):
        """Test that normalization happens during ingest."""
        content = """HELLO
World
TESTING
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".txt", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = PlainTextIngestor(language="en")
            result = ingestor.ingest(filepath)

            normalized = [w.normalized for w in result.words]
            assert all(w.islower() for w in normalized)
            assert "hello" in normalized
            assert "world" in normalized
        finally:
            filepath.unlink()

    def test_duplicate_detection(self):
        """Test that duplicates are detected and merged."""
        content = """hello
HELLO
Hello
HeLLo
"""
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".txt", delete=False, encoding="utf-8"
        ) as f:
            f.write(content)
            filepath = Path(f.name)

        try:
            ingestor = PlainTextIngestor(language="en")
            result = ingestor.ingest(filepath)

            # All should normalize to "hello"
            assert result.total_valid == 1
            assert result.total_duplicates == 3
            assert result.words[0].normalized == "hello"
            assert len(result.words[0].sources) == 4
        finally:
            filepath.unlink()
