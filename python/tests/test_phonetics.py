"""Tests for the phonetics module."""

import pytest

from ditong.phonetics.transcribe import (
    Transcriber,
    EpitranTranscriber,
    EspeakTranscriber,
    PhonemizerTranscriber,
    G2PEnglishTranscriber,
    GruutTranscriber,
    get_transcriber,
    register_transcriber,
    list_transcribers,
    get_default_transcriber,
    set_default_transcriber,
    transcribe,
    batch_transcribe,
    _TRANSCRIBERS,
    _init_registry,
)


class TestTranscriberRegistry:
    """Tests for transcriber registry functions."""

    def test_list_transcribers(self):
        """Test listing available transcribers."""
        transcribers = list_transcribers()
        assert "epitran" in transcribers
        assert "espeak" in transcribers
        assert "phonemizer" in transcribers
        assert "g2p_en" in transcribers
        assert "gruut" in transcribers

    def test_get_transcriber(self):
        """Test getting transcriber by name."""
        transcriber = get_transcriber("epitran")
        assert isinstance(transcriber, EpitranTranscriber)

    def test_get_transcriber_cached(self):
        """Test that transcribers are cached."""
        t1 = get_transcriber("epitran")
        t2 = get_transcriber("epitran")
        assert t1 is t2

    def test_get_transcriber_unknown(self):
        """Test error on unknown transcriber."""
        with pytest.raises(ValueError) as exc_info:
            get_transcriber("nonexistent")
        assert "Unknown transcriber" in str(exc_info.value)

    def test_register_transcriber(self):
        """Test registering custom transcriber."""

        class CustomTranscriber(Transcriber):
            name = "custom"
            supported_languages = ["en"]

            def transcribe(self, word: str, language: str):
                return f"custom:{word}"

        register_transcriber("custom_test", CustomTranscriber)
        assert "custom_test" in list_transcribers()

        transcriber = get_transcriber("custom_test")
        assert transcriber.transcribe("hello", "en") == "custom:hello"

    def test_default_transcriber(self):
        """Test default transcriber setting."""
        original = get_default_transcriber()
        assert original == "epitran"

        set_default_transcriber("espeak")
        assert get_default_transcriber() == "espeak"

        # Reset
        set_default_transcriber(original)

    def test_set_default_transcriber_unknown(self):
        """Test error on setting unknown default."""
        with pytest.raises(ValueError):
            set_default_transcriber("nonexistent")


class TestEpitranTranscriber:
    """Tests for EpitranTranscriber."""

    def test_supported_languages(self):
        """Test supported language list."""
        t = EpitranTranscriber()
        assert "en" in t.supported_languages
        assert "tr" in t.supported_languages
        assert "de" in t.supported_languages

    def test_supports_language(self):
        """Test language support check."""
        t = EpitranTranscriber()
        assert t.supports_language("en") is True
        assert t.supports_language("xx") is False

    def test_lang_map(self):
        """Test language code mapping."""
        assert EpitranTranscriber._LANG_MAP["en"] == "eng-Latn"
        assert EpitranTranscriber._LANG_MAP["tr"] == "tur-Latn"


class TestEspeakTranscriber:
    """Tests for EspeakTranscriber."""

    def test_supported_languages(self):
        """Test supported language list."""
        t = EspeakTranscriber()
        assert "en" in t.supported_languages
        assert "tr" in t.supported_languages

    def test_lang_map(self):
        """Test language code mapping."""
        assert EspeakTranscriber._LANG_MAP["en"] == "en-us"
        assert EspeakTranscriber._LANG_MAP["tr"] == "tr"


class TestG2PEnglishTranscriber:
    """Tests for G2PEnglishTranscriber."""

    def test_english_only(self):
        """Test that only English is supported."""
        t = G2PEnglishTranscriber()
        assert t.supported_languages == ["en"]
        assert t.supports_language("en") is True
        assert t.supports_language("tr") is False

    def test_transcribe_non_english(self):
        """Test that non-English returns None."""
        t = G2PEnglishTranscriber()
        result = t.transcribe("hello", "tr")
        assert result is None


class TestTranscriberBase:
    """Tests for Transcriber base class."""

    def test_batch_transcribe(self):
        """Test batch transcription."""

        class MockTranscriber(Transcriber):
            name = "mock"
            supported_languages = ["en"]

            def transcribe(self, word: str, language: str):
                return f"ipa:{word}"

        t = MockTranscriber()
        results = t.batch_transcribe(["hello", "world"], "en")

        assert results["hello"] == "ipa:hello"
        assert results["world"] == "ipa:world"

    def test_batch_transcribe_skip_errors(self):
        """Test batch transcription with error skipping."""

        class FailingTranscriber(Transcriber):
            name = "failing"
            supported_languages = ["en"]

            def transcribe(self, word: str, language: str):
                if word == "fail":
                    raise ValueError("Intentional failure")
                return f"ipa:{word}"

        t = FailingTranscriber()
        results = t.batch_transcribe(["hello", "fail", "world"], "en", skip_errors=True)

        assert results["hello"] == "ipa:hello"
        assert results["fail"] is None
        assert results["world"] == "ipa:world"

    def test_batch_transcribe_raise_errors(self):
        """Test batch transcription raising errors."""

        class FailingTranscriber(Transcriber):
            name = "failing2"
            supported_languages = ["en"]

            def transcribe(self, word: str, language: str):
                if word == "fail":
                    raise ValueError("Intentional failure")
                return f"ipa:{word}"

        t = FailingTranscriber()
        with pytest.raises(ValueError):
            t.batch_transcribe(["hello", "fail", "world"], "en", skip_errors=False)


class TestConvenienceFunctions:
    """Tests for module-level convenience functions."""

    def test_transcribe_uses_default(self):
        """Test that transcribe uses default backend."""
        # This test may fail if epitran is not installed
        # In that case, it will return None
        result = transcribe("hello", "en")
        # Just verify it doesn't crash
        assert result is None or isinstance(result, str)

    def test_transcribe_with_backend(self):
        """Test transcribe with explicit backend."""
        # Register a mock for testing
        class MockTranscriber(Transcriber):
            name = "mock_for_test"
            supported_languages = ["en"]

            def transcribe(self, word: str, language: str):
                return f"mock:{word}"

        register_transcriber("mock_for_test", MockTranscriber)

        result = transcribe("hello", "en", backend="mock_for_test")
        assert result == "mock:hello"

    def test_batch_transcribe_function(self):
        """Test batch_transcribe module function."""

        class MockTranscriber(Transcriber):
            name = "mock_batch"
            supported_languages = ["en"]

            def transcribe(self, word: str, language: str):
                return f"batch:{word}"

        register_transcriber("mock_batch", MockTranscriber)

        results = batch_transcribe(["hello", "world"], "en", backend="mock_batch")
        assert results["hello"] == "batch:hello"
        assert results["world"] == "batch:world"


class TestTranscriberIntegration:
    """Integration tests (may require external dependencies)."""

    @pytest.mark.skip(reason="Requires epitran installation")
    def test_epitran_real_transcription(self):
        """Test real epitran transcription."""
        t = get_transcriber("epitran")
        result = t.transcribe("hello", "en")
        assert result is not None
        assert len(result) > 0

    @pytest.mark.skip(reason="Requires espeak-ng installation")
    def test_espeak_real_transcription(self):
        """Test real espeak transcription."""
        t = get_transcriber("espeak")
        result = t.transcribe("hello", "en")
        assert result is not None
