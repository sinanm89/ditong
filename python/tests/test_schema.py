"""Tests for the schema module."""

import pytest
import json
import tempfile
from pathlib import Path
from ditong.schema import Word, WordSource, WordType, Dictionary


class TestWordType:
    """Tests for WordType enum."""

    def test_from_length_valid(self):
        """Test valid length conversions."""
        assert WordType.from_length(3) == WordType.C3
        assert WordType.from_length(4) == WordType.C4
        assert WordType.from_length(5) == WordType.C5
        assert WordType.from_length(6) == WordType.C6
        assert WordType.from_length(7) == WordType.C7
        assert WordType.from_length(8) == WordType.C8
        assert WordType.from_length(9) == WordType.C9
        assert WordType.from_length(10) == WordType.C10

    def test_from_length_invalid(self):
        """Test invalid lengths return None."""
        assert WordType.from_length(0) is None
        assert WordType.from_length(1) is None
        assert WordType.from_length(2) is None
        assert WordType.from_length(11) is None
        assert WordType.from_length(100) is None

    def test_word_type_values(self):
        """Test WordType enum values."""
        assert WordType.C5.value == "5-c"
        assert WordType.C8.value == "8-c"


class TestWordSource:
    """Tests for WordSource dataclass."""

    def test_creation(self):
        """Test WordSource creation."""
        source = WordSource(
            dict_name="hunspell_en",
            dict_filepath="/path/to/en.dic",
            language="en",
            original_form="hello",
            line_number=123,
            category="standard",
        )
        assert source.dict_name == "hunspell_en"
        assert source.language == "en"
        assert source.original_form == "hello"
        assert source.line_number == 123

    def test_to_dict(self):
        """Test WordSource serialization."""
        source = WordSource(
            dict_name="hunspell_en",
            dict_filepath="/path/to/en.dic",
            language="en",
            original_form="hello",
            line_number=123,
            category="standard",
        )
        d = source.to_dict()
        assert d["dict_name"] == "hunspell_en"
        assert d["language"] == "en"
        assert d["line_number"] == 123

    def test_from_dict(self):
        """Test WordSource deserialization."""
        data = {
            "dict_name": "hunspell_tr",
            "dict_filepath": "/path/to/tr.dic",
            "language": "tr",
            "original_form": "çare",
            "line_number": 456,
            "category": "standard",
        }
        source = WordSource.from_dict(data)
        assert source.dict_name == "hunspell_tr"
        assert source.original_form == "çare"

    def test_default_category(self):
        """Test default category is standard."""
        source = WordSource(
            dict_name="test",
            dict_filepath="/test",
            language="en",
            original_form="test",
        )
        assert source.category == "standard"


class TestWord:
    """Tests for Word dataclass."""

    def test_creation(self):
        """Test Word creation."""
        word = Word(
            normalized="hello",
            length=5,
            word_type="5-c",
        )
        assert word.normalized == "hello"
        assert word.length == 5
        assert word.word_type == "5-c"
        assert word.sources == []
        assert word.categories == set()

    def test_add_source(self):
        """Test adding sources to a word."""
        word = Word(normalized="care", length=4, word_type="4-c")

        source_en = WordSource(
            dict_name="hunspell_en",
            dict_filepath="/en.dic",
            language="en",
            original_form="care",
            category="standard",
        )
        source_tr = WordSource(
            dict_name="hunspell_tr",
            dict_filepath="/tr.dic",
            language="tr",
            original_form="çare",
            category="standard",
        )

        word.add_source(source_en)
        word.add_source(source_tr)

        assert len(word.sources) == 2
        assert "en" in word.languages
        assert "tr" in word.languages
        assert "standard" in word.categories

    def test_get_source_dicts(self):
        """Test getting source dictionary names."""
        word = Word(normalized="test", length=4, word_type="4-c")
        word.add_source(WordSource(
            dict_name="dict1",
            dict_filepath="/d1",
            language="en",
            original_form="test",
        ))
        word.add_source(WordSource(
            dict_name="dict2",
            dict_filepath="/d2",
            language="tr",
            original_form="test",
        ))

        dicts = word.get_source_dicts()
        assert "dict1" in dicts
        assert "dict2" in dicts

    def test_matches_filter_length(self):
        """Test word filtering by length."""
        word = Word(normalized="hello", length=5, word_type="5-c")

        assert word.matches_filter(min_length=3, max_length=8) is True
        assert word.matches_filter(min_length=5, max_length=5) is True
        assert word.matches_filter(min_length=6, max_length=8) is False
        assert word.matches_filter(min_length=3, max_length=4) is False

    def test_matches_filter_language(self):
        """Test word filtering by language."""
        word = Word(normalized="test", length=4, word_type="4-c")
        word.add_source(WordSource(
            dict_name="d", dict_filepath="/d", language="en", original_form="test"
        ))

        assert word.matches_filter(include_languages={"en"}) is True
        assert word.matches_filter(include_languages={"tr"}) is False
        assert word.matches_filter(include_languages={"en", "tr"}) is True

    def test_matches_filter_category(self):
        """Test word filtering by category."""
        word = Word(normalized="test", length=4, word_type="4-c")
        word.add_source(WordSource(
            dict_name="d", dict_filepath="/d", language="en",
            original_form="test", category="standard"
        ))

        assert word.matches_filter(include_categories={"standard"}) is True
        assert word.matches_filter(include_categories={"urban"}) is False
        assert word.matches_filter(exclude_categories={"urban"}) is True
        assert word.matches_filter(exclude_categories={"standard"}) is False

    def test_to_dict(self):
        """Test Word serialization."""
        word = Word(normalized="hello", length=5, word_type="5-c")
        word.add_source(WordSource(
            dict_name="test",
            dict_filepath="/test",
            language="en",
            original_form="hello",
        ))

        d = word.to_dict()
        assert d["normalized"] == "hello"
        assert d["length"] == 5
        assert d["type"] == "5-c"
        assert len(d["sources"]) == 1

    def test_from_dict(self):
        """Test Word deserialization."""
        data = {
            "normalized": "hello",
            "length": 5,
            "type": "5-c",
            "sources": [{
                "dict_name": "test",
                "dict_filepath": "/test",
                "language": "en",
                "original_form": "hello",
                "category": "standard",
            }],
            "categories": ["standard"],
            "languages": ["en"],
            "tags": [],
        }
        word = Word.from_dict(data)
        assert word.normalized == "hello"
        assert word.length == 5
        assert len(word.sources) == 1


class TestDictionary:
    """Tests for Dictionary dataclass."""

    def test_creation(self):
        """Test Dictionary creation."""
        d = Dictionary(name="test_dict")
        assert d.name == "test_dict"
        assert d.count() == 0

    def test_add_word(self):
        """Test adding words to dictionary."""
        d = Dictionary(name="test")
        word = Word(normalized="hello", length=5, word_type="5-c")

        d.add_word(word)
        assert d.count() == 1
        assert "hello" in d.words

    def test_add_word_merge(self):
        """Test merging duplicate words."""
        d = Dictionary(name="test")

        word1 = Word(normalized="care", length=4, word_type="4-c")
        word1.add_source(WordSource(
            dict_name="en", dict_filepath="/en", language="en", original_form="care"
        ))

        word2 = Word(normalized="care", length=4, word_type="4-c")
        word2.add_source(WordSource(
            dict_name="tr", dict_filepath="/tr", language="tr", original_form="çare"
        ))

        d.add_word(word1)
        d.add_word(word2)

        assert d.count() == 1
        assert len(d.words["care"].sources) == 2

    def test_get_words_sorted(self):
        """Test getting sorted word list."""
        d = Dictionary(name="test")
        d.add_word(Word(normalized="zebra", length=5, word_type="5-c"))
        d.add_word(Word(normalized="apple", length=5, word_type="5-c"))
        d.add_word(Word(normalized="mango", length=5, word_type="5-c"))

        words = d.get_words_list()
        assert words[0].normalized == "apple"
        assert words[1].normalized == "mango"
        assert words[2].normalized == "zebra"

    def test_save_and_load(self):
        """Test saving and loading dictionary."""
        d = Dictionary(name="test_save")
        word = Word(normalized="hello", length=5, word_type="5-c")
        word.add_source(WordSource(
            dict_name="test", dict_filepath="/test", language="en", original_form="hello"
        ))
        d.add_word(word)

        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            filepath = Path(f.name)

        try:
            d.save(filepath)
            loaded = Dictionary.load(filepath)

            assert loaded.name == "test_save"
            assert loaded.count() == 1
            assert "hello" in loaded.words
        finally:
            filepath.unlink()

    def test_to_dict(self):
        """Test Dictionary serialization."""
        d = Dictionary(name="test", language="en", word_type="5-c")
        d.add_word(Word(normalized="hello", length=5, word_type="5-c"))

        data = d.to_dict()
        assert data["name"] == "test"
        assert data["language"] == "en"
        assert data["word_count"] == 1
        assert "hello" in data["words"]
