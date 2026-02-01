"""Dictionary builder for organized output.

Creates per-language, per-length JSON files with full word metadata.

Output structure:
    dicts/
    ├── en/
    │   ├── 3-c.json
    │   ├── 4-c.json
    │   ├── 5-c.json
    │   └── ...
    └── tr/
        ├── 3-c.json
        └── ...
"""

from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional
import json

from ..schema import Word, Dictionary
from ..ingest.base import IngestResult


@dataclass
class BuildStats:
    """Statistics from a build operation."""

    total_words: int = 0
    by_length: dict[int, int] = field(default_factory=dict)
    by_language: dict[str, int] = field(default_factory=dict)
    by_category: dict[str, int] = field(default_factory=dict)
    files_written: list[str] = field(default_factory=list)


class DictionaryBuilder:
    """Builds organized dictionary files from ingested words."""

    def __init__(
        self,
        output_dir: Path | str,
        min_length: int = 3,
        max_length: int = 10,
    ):
        """Initialize builder.

        Args:
            output_dir: Base directory for output files.
            min_length: Minimum word length to include.
            max_length: Maximum word length to include.
        """
        self.output_dir = Path(output_dir)
        self.min_length = min_length
        self.max_length = max_length

        # Internal word storage: language -> length -> normalized -> Word
        self._words: dict[str, dict[int, dict[str, Word]]] = {}

    def add_words(self, result: IngestResult) -> None:
        """Add words from an IngestResult.

        Args:
            result: IngestResult from an ingestor.
        """
        language = result.language

        if language not in self._words:
            self._words[language] = {
                i: {} for i in range(self.min_length, self.max_length + 1)
            }

        for word in result.words:
            if word.length < self.min_length or word.length > self.max_length:
                continue

            length_dict = self._words[language].get(word.length)
            if length_dict is None:
                continue

            if word.normalized in length_dict:
                # Merge sources
                existing = length_dict[word.normalized]
                for source in word.sources:
                    existing.add_source(source)
                existing.tags.update(word.tags)
            else:
                length_dict[word.normalized] = word

    def add_word(self, word: Word, language: str) -> None:
        """Add a single word.

        Args:
            word: Word to add.
            language: Language code.
        """
        if language not in self._words:
            self._words[language] = {
                i: {} for i in range(self.min_length, self.max_length + 1)
            }

        if word.length < self.min_length or word.length > self.max_length:
            return

        length_dict = self._words[language].get(word.length)
        if length_dict is None:
            return

        if word.normalized in length_dict:
            existing = length_dict[word.normalized]
            for source in word.sources:
                existing.add_source(source)
        else:
            length_dict[word.normalized] = word

    def get_languages(self) -> list[str]:
        """Get list of languages with words."""
        return list(self._words.keys())

    def get_word_count(
        self,
        language: Optional[str] = None,
        length: Optional[int] = None
    ) -> int:
        """Get word count, optionally filtered."""
        count = 0

        languages = [language] if language else self._words.keys()
        for lang in languages:
            if lang not in self._words:
                continue

            if length is not None:
                if length in self._words[lang]:
                    count += len(self._words[lang][length])
            else:
                for length_dict in self._words[lang].values():
                    count += len(length_dict)

        return count

    def build(self) -> BuildStats:
        """Build all dictionary files.

        Returns:
            BuildStats with counts and file paths.
        """
        stats = BuildStats()

        for language, length_dicts in self._words.items():
            lang_dir = self.output_dir / language
            lang_dir.mkdir(parents=True, exist_ok=True)

            for length, words_dict in length_dicts.items():
                if not words_dict:
                    continue

                # Create Dictionary object
                word_type = f"{length}-c"
                dictionary = Dictionary(
                    name=f"{language}_{word_type}",
                    language=language,
                    word_type=word_type,
                )

                for word in words_dict.values():
                    dictionary.add_word(word)
                    stats.total_words += 1
                    stats.by_length[length] = stats.by_length.get(length, 0) + 1
                    stats.by_language[language] = (
                        stats.by_language.get(language, 0) + 1
                    )
                    for cat in word.categories:
                        stats.by_category[cat] = stats.by_category.get(cat, 0) + 1

                # Save to file
                filepath = lang_dir / f"{word_type}.json"
                dictionary.save(filepath)
                stats.files_written.append(str(filepath))

        return stats

    def build_combined(self, name: str = "all") -> BuildStats:
        """Build a combined dictionary with all languages merged.

        Words that normalize the same across languages are merged.

        Args:
            name: Name for the combined dictionary.

        Returns:
            BuildStats.
        """
        stats = BuildStats()

        # Merge all words by normalized form
        combined: dict[int, dict[str, Word]] = {
            i: {} for i in range(self.min_length, self.max_length + 1)
        }

        for language, length_dicts in self._words.items():
            for length, words_dict in length_dicts.items():
                for normalized, word in words_dict.items():
                    if normalized in combined[length]:
                        existing = combined[length][normalized]
                        for source in word.sources:
                            existing.add_source(source)
                        existing.tags.update(word.tags)
                    else:
                        # Clone the word
                        combined[length][normalized] = Word(
                            normalized=word.normalized,
                            length=word.length,
                            word_type=word.word_type,
                            sources=list(word.sources),
                            ipa=word.ipa,
                        )
                        combined[length][normalized].categories = set(word.categories)
                        combined[length][normalized].languages = set(word.languages)
                        combined[length][normalized].tags = set(word.tags)

        # Write combined files
        combined_dir = self.output_dir / name
        combined_dir.mkdir(parents=True, exist_ok=True)

        for length, words_dict in combined.items():
            if not words_dict:
                continue

            word_type = f"{length}-c"
            dictionary = Dictionary(
                name=f"{name}_{word_type}",
                word_type=word_type,
            )

            for word in words_dict.values():
                dictionary.add_word(word)
                stats.total_words += 1
                stats.by_length[length] = stats.by_length.get(length, 0) + 1
                for lang in word.languages:
                    stats.by_language[lang] = stats.by_language.get(lang, 0) + 1

            filepath = combined_dir / f"{word_type}.json"
            dictionary.save(filepath)
            stats.files_written.append(str(filepath))

        return stats
