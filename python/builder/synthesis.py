"""Synthesis builder for cross-language word unions.

Creates filtered word sets based on configurable criteria:
- Include/exclude languages
- Include/exclude categories (standard, urban, curseword, etc.)
- Length constraints
- Split by first letter for large outputs

Output structure:
    dicts/synthesis/
    ├── en_tr_standard/
    │   ├── 5-c/
    │   │   ├── a.json
    │   │   ├── b.json
    │   │   └── ...
    │   └── 6-c/
    │       └── ...
    └── en_tr_de_no_urban/
        └── ...
"""

from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional
import json

from ..schema import Word, Dictionary


@dataclass
class SynthesisConfig:
    """Configuration for a synthesis dictionary.

    Examples:
        # EN + TR standard words only
        SynthesisConfig(
            name="en_tr_standard",
            include_languages={"en", "tr"},
            include_categories={"standard"},
        )

        # All languages, no urban/curse words
        SynthesisConfig(
            name="all_clean",
            exclude_categories={"urban", "curseword"},
        )

        # EN + TR + DE, 5-6 char only
        SynthesisConfig(
            name="en_tr_de_short",
            include_languages={"en", "tr", "de"},
            min_length=5,
            max_length=6,
        )
    """

    name: str
    include_languages: Optional[set[str]] = None   # None = all
    exclude_languages: Optional[set[str]] = None
    include_categories: Optional[set[str]] = None  # None = all
    exclude_categories: Optional[set[str]] = None
    min_length: int = 3
    max_length: int = 10
    split_by_letter: bool = True  # Split large outputs by first letter

    def matches_word(self, word: Word) -> bool:
        """Check if word matches this config's filters."""
        # Length check
        if word.length < self.min_length or word.length > self.max_length:
            return False

        # Language check
        if self.include_languages:
            if not word.languages & self.include_languages:
                return False
        if self.exclude_languages:
            if word.languages & self.exclude_languages:
                return False

        # Category check
        if self.include_categories:
            if not word.categories & self.include_categories:
                return False
        if self.exclude_categories:
            if word.categories & self.exclude_categories:
                return False

        return True

    def to_dict(self) -> dict:
        """Serialize config for metadata."""
        return {
            "name": self.name,
            "include_languages": sorted(self.include_languages) if self.include_languages else None,
            "exclude_languages": sorted(self.exclude_languages) if self.exclude_languages else None,
            "include_categories": sorted(self.include_categories) if self.include_categories else None,
            "exclude_categories": sorted(self.exclude_categories) if self.exclude_categories else None,
            "min_length": self.min_length,
            "max_length": self.max_length,
        }


@dataclass
class SynthesisStats:
    """Statistics from a synthesis build."""

    config_name: str
    total_words: int = 0
    by_length: dict[int, int] = field(default_factory=dict)
    by_letter: dict[str, int] = field(default_factory=dict)
    languages_included: set[str] = field(default_factory=set)
    categories_included: set[str] = field(default_factory=set)
    files_written: list[str] = field(default_factory=list)


class SynthesisBuilder:
    """Builds synthesis dictionaries from word pools."""

    def __init__(self, output_dir: Path | str):
        """Initialize builder.

        Args:
            output_dir: Base directory for synthesis output.
        """
        self.output_dir = Path(output_dir) / "synthesis"
        self._word_pool: dict[str, Word] = {}  # normalized -> Word

    def add_words(self, words: list[Word]) -> None:
        """Add words to the pool for synthesis.

        Args:
            words: List of words to add.
        """
        for word in words:
            if word.normalized in self._word_pool:
                # Merge sources
                existing = self._word_pool[word.normalized]
                for source in word.sources:
                    existing.add_source(source)
                existing.tags.update(word.tags)
            else:
                self._word_pool[word.normalized] = word

    def clear_pool(self) -> None:
        """Clear the word pool."""
        self._word_pool.clear()

    def get_pool_size(self) -> int:
        """Get number of words in pool."""
        return len(self._word_pool)

    def build(self, config: SynthesisConfig) -> SynthesisStats:
        """Build a synthesis dictionary based on config.

        Args:
            config: SynthesisConfig defining filters.

        Returns:
            SynthesisStats.
        """
        stats = SynthesisStats(config_name=config.name)

        # Filter words
        filtered: dict[int, dict[str, list[Word]]] = {}
        for length in range(config.min_length, config.max_length + 1):
            filtered[length] = {}

        for word in self._word_pool.values():
            if not config.matches_word(word):
                continue

            letter = word.normalized[0] if word.normalized else "_"
            if letter not in filtered[word.length]:
                filtered[word.length][letter] = []

            filtered[word.length][letter].append(word)
            stats.total_words += 1
            stats.by_length[word.length] = stats.by_length.get(word.length, 0) + 1
            stats.by_letter[letter] = stats.by_letter.get(letter, 0) + 1
            stats.languages_included.update(word.languages)
            stats.categories_included.update(word.categories)

        # Write output
        synth_dir = self.output_dir / config.name
        synth_dir.mkdir(parents=True, exist_ok=True)

        # Write config metadata
        config_file = synth_dir / "_config.json"
        with open(config_file, "w", encoding="utf-8") as f:
            json.dump({
                "config": config.to_dict(),
                "generated_at": datetime.now(timezone.utc).isoformat(),
                "stats": {
                    "total_words": stats.total_words,
                    "by_length": {str(k): v for k, v in stats.by_length.items()},
                    "languages": sorted(stats.languages_included),
                    "categories": sorted(stats.categories_included),
                },
            }, f, indent=2)
        stats.files_written.append(str(config_file))

        # Write word files
        for length, letter_dict in filtered.items():
            if not letter_dict:
                continue

            word_type = f"{length}-c"

            if config.split_by_letter:
                # Split by first letter
                length_dir = synth_dir / word_type
                length_dir.mkdir(parents=True, exist_ok=True)

                for letter, words in sorted(letter_dict.items()):
                    dictionary = Dictionary(
                        name=f"{config.name}_{word_type}_{letter}",
                        word_type=word_type,
                    )
                    for word in sorted(words, key=lambda w: w.normalized):
                        dictionary.add_word(word)

                    filepath = length_dir / f"{letter}.json"
                    dictionary.save(filepath)
                    stats.files_written.append(str(filepath))
            else:
                # Single file per length
                dictionary = Dictionary(
                    name=f"{config.name}_{word_type}",
                    word_type=word_type,
                )
                for words in letter_dict.values():
                    for word in words:
                        dictionary.add_word(word)

                filepath = synth_dir / f"{word_type}.json"
                dictionary.save(filepath)
                stats.files_written.append(str(filepath))

        return stats

    def build_multiple(self, configs: list[SynthesisConfig]) -> list[SynthesisStats]:
        """Build multiple synthesis dictionaries.

        Args:
            configs: List of SynthesisConfig objects.

        Returns:
            List of SynthesisStats.
        """
        return [self.build(config) for config in configs]
