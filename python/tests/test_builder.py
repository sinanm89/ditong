"""Tests for the builder module."""

import pytest
import tempfile
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

from schema import Word, WordSource
from builder.dictionary import DictionaryBuilder
from builder.synthesis import SynthesisBuilder, SynthesisConfig


def create_test_word(normalized: str, language: str, category: str = "standard") -> Word:
    """Helper to create a test word."""
    word = Word(
        normalized=normalized,
        length=len(normalized),
        word_type=f"{len(normalized)}-c",
    )
    word.add_source(WordSource(
        dict_name=f"test_{language}",
        dict_filepath=f"/test/{language}.dic",
        language=language,
        original_form=normalized,
        category=category,
    ))
    return word


class TestDictionaryBuilder:
    """Tests for DictionaryBuilder."""

    def test_add_words(self):
        """Test adding words to builder."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = DictionaryBuilder(
                output_dir=tmpdir,
                min_length=3,
                max_length=10,
            )

            from ingest.base import IngestResult
            result = IngestResult(
                words=[
                    create_test_word("hello", "en"),
                    create_test_word("world", "en"),
                ],
                source_path="/test",
                dict_name="test",
                language="en",
                category="standard",
            )

            builder.add_words(result)
            assert builder.get_word_count(language="en") == 2

    def test_get_word_count(self):
        """Test word count methods."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = DictionaryBuilder(tmpdir, 3, 10)

            builder.add_word(create_test_word("hello", "en"), "en")
            builder.add_word(create_test_word("world", "en"), "en")
            builder.add_word(create_test_word("test", "tr"), "tr")

            assert builder.get_word_count() == 3
            assert builder.get_word_count(language="en") == 2
            assert builder.get_word_count(language="tr") == 1
            assert builder.get_word_count(language="en", length=5) == 2
            assert builder.get_word_count(language="en", length=4) == 0

    def test_build_creates_files(self):
        """Test that build creates correct file structure."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = DictionaryBuilder(tmpdir, 4, 6)

            builder.add_word(create_test_word("test", "en"), "en")  # 4 chars
            builder.add_word(create_test_word("hello", "en"), "en")  # 5 chars
            builder.add_word(create_test_word("worlds", "en"), "en")  # 6 chars

            stats = builder.build()

            assert stats.total_words == 3
            assert len(stats.files_written) == 3

            # Check files exist
            assert Path(tmpdir, "en", "4-c.json").exists()
            assert Path(tmpdir, "en", "5-c.json").exists()
            assert Path(tmpdir, "en", "6-c.json").exists()

    def test_build_json_content(self):
        """Test that built JSON has correct structure."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = DictionaryBuilder(tmpdir, 5, 5)
            builder.add_word(create_test_word("hello", "en"), "en")
            builder.build()

            json_path = Path(tmpdir, "en", "5-c.json")
            with open(json_path) as f:
                data = json.load(f)

            assert data["name"] == "en_5-c"
            assert data["language"] == "en"
            assert data["word_type"] == "5-c"
            assert "hello" in data["words"]
            assert data["words"]["hello"]["normalized"] == "hello"

    def test_build_combined(self):
        """Test building combined dictionary."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = DictionaryBuilder(tmpdir, 4, 5)

            # Add "care" from both languages
            word_en = create_test_word("care", "en")
            word_tr = Word(normalized="care", length=4, word_type="4-c")
            word_tr.add_source(WordSource(
                dict_name="test_tr",
                dict_filepath="/tr.dic",
                language="tr",
                original_form="çare",
                category="standard",
            ))

            builder.add_word(word_en, "en")
            builder.add_word(word_tr, "tr")

            stats = builder.build_combined(name="all")

            # Should have 1 unique word
            json_path = Path(tmpdir, "all", "4-c.json")
            with open(json_path) as f:
                data = json.load(f)

            assert "care" in data["words"]
            # Should have both languages
            assert "en" in data["words"]["care"]["languages"]
            assert "tr" in data["words"]["care"]["languages"]


class TestSynthesisConfig:
    """Tests for SynthesisConfig."""

    def test_matches_word_length(self):
        """Test length filtering."""
        config = SynthesisConfig(name="test", min_length=5, max_length=7)

        word4 = create_test_word("test", "en")  # 4 chars
        word5 = create_test_word("hello", "en")  # 5 chars
        word8 = create_test_word("greeting", "en")  # 8 chars

        assert config.matches_word(word4) is False
        assert config.matches_word(word5) is True
        assert config.matches_word(word8) is False

    def test_matches_word_language(self):
        """Test language filtering."""
        config = SynthesisConfig(
            name="test",
            include_languages={"en", "tr"},
            min_length=3,
            max_length=10,
        )

        word_en = create_test_word("hello", "en")
        word_de = create_test_word("hallo", "de")

        assert config.matches_word(word_en) is True
        assert config.matches_word(word_de) is False

    def test_matches_word_category(self):
        """Test category filtering."""
        config = SynthesisConfig(
            name="test",
            include_categories={"standard"},
            exclude_categories={"curseword"},
            min_length=3,
            max_length=10,
        )

        word_std = create_test_word("hello", "en", category="standard")
        word_curse = create_test_word("badword", "en", category="curseword")

        assert config.matches_word(word_std) is True
        assert config.matches_word(word_curse) is False

    def test_to_dict(self):
        """Test config serialization."""
        config = SynthesisConfig(
            name="en_tr_test",
            include_languages={"en", "tr"},
            min_length=5,
            max_length=8,
        )

        data = config.to_dict()
        assert data["name"] == "en_tr_test"
        assert set(data["include_languages"]) == {"en", "tr"}
        assert data["min_length"] == 5


class TestSynthesisBuilder:
    """Tests for SynthesisBuilder."""

    def test_add_words(self):
        """Test adding words to pool."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = SynthesisBuilder(tmpdir)

            words = [
                create_test_word("hello", "en"),
                create_test_word("world", "en"),
            ]
            builder.add_words(words)

            assert builder.get_pool_size() == 2

    def test_add_words_merge(self):
        """Test that duplicate words are merged."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = SynthesisBuilder(tmpdir)

            word1 = create_test_word("care", "en")
            word2 = Word(normalized="care", length=4, word_type="4-c")
            word2.add_source(WordSource(
                dict_name="tr",
                dict_filepath="/tr.dic",
                language="tr",
                original_form="çare",
                category="standard",
            ))

            builder.add_words([word1])
            builder.add_words([word2])

            assert builder.get_pool_size() == 1

    def test_build_creates_files(self):
        """Test that build creates synthesis files."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = SynthesisBuilder(tmpdir)

            words = [
                create_test_word("alpha", "en"),
                create_test_word("apple", "en"),
                create_test_word("beta", "en"),
            ]
            builder.add_words(words)

            config = SynthesisConfig(
                name="test_synth",
                include_languages={"en"},
                min_length=4,
                max_length=6,
                split_by_letter=True,
            )

            stats = builder.build(config)

            assert stats.total_words == 3
            assert stats.config_name == "test_synth"

            # Check config file
            config_file = Path(tmpdir, "synthesis", "test_synth", "_config.json")
            assert config_file.exists()

            # Check letter-split files
            a_file = Path(tmpdir, "synthesis", "test_synth", "5-c", "a.json")
            b_file = Path(tmpdir, "synthesis", "test_synth", "4-c", "b.json")
            assert a_file.exists()
            assert b_file.exists()

    def test_build_no_split(self):
        """Test building without letter splitting."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = SynthesisBuilder(tmpdir)
            builder.add_words([create_test_word("hello", "en")])

            config = SynthesisConfig(
                name="no_split",
                min_length=5,
                max_length=5,
                split_by_letter=False,
            )

            stats = builder.build(config)

            # Should have single file per length, not per letter
            json_file = Path(tmpdir, "synthesis", "no_split", "5-c.json")
            assert json_file.exists()

    def test_build_filters_correctly(self):
        """Test that build applies filters."""
        with tempfile.TemporaryDirectory() as tmpdir:
            builder = SynthesisBuilder(tmpdir)

            builder.add_words([
                create_test_word("hello", "en", "standard"),
                create_test_word("curse", "en", "curseword"),
                create_test_word("merhaba", "tr", "standard"),
            ])

            config = SynthesisConfig(
                name="filtered",
                include_languages={"en"},
                include_categories={"standard"},
                exclude_categories={"curseword"},
                min_length=3,
                max_length=10,
            )

            stats = builder.build(config)

            # Should only include "hello" (en, standard)
            assert stats.total_words == 1
            assert "en" in stats.languages_included
            assert "tr" not in stats.languages_included
