"""Word schema and data structures for ditong.

Core concept:
    - Words from multiple sources normalize to the same form
    - Each word tracks its origin sources and metadata
    - Synthesis configs filter words by source tags

Example:
    "care" (EN) + "çare" (TR) → normalized "care"
    Tagged with sources: ["hunspell_en_us", "hunspell_tr"]
"""

from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from pathlib import Path
from typing import Any, Optional
import json


class WordType(Enum):
    """Word length type identifier."""

    C3 = "3-c"
    C4 = "4-c"
    C5 = "5-c"
    C6 = "6-c"
    C7 = "7-c"
    C8 = "8-c"
    C9 = "9-c"
    C10 = "10-c"

    @classmethod
    def from_length(cls, length: int) -> Optional["WordType"]:
        """Get WordType from character length."""
        mapping = {
            3: cls.C3, 4: cls.C4, 5: cls.C5, 6: cls.C6,
            7: cls.C7, 8: cls.C8, 9: cls.C9, 10: cls.C10,
        }
        return mapping.get(length)


@dataclass
class WordSource:
    """Tracks where a word came from."""

    dict_name: str          # e.g., "hunspell_en_us", "urban_dictionary"
    dict_filepath: str      # Full path to source file
    language: str           # e.g., "en", "tr"
    original_form: str      # Original word before normalization
    line_number: Optional[int] = None  # Line in source file (if applicable)
    category: str = "standard"  # e.g., "standard", "urban", "curseword"

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "dict_name": self.dict_name,
            "dict_filepath": self.dict_filepath,
            "language": self.language,
            "original_form": self.original_form,
            "line_number": self.line_number,
            "category": self.category,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "WordSource":
        """Create from dictionary."""
        return cls(
            dict_name=data["dict_name"],
            dict_filepath=data["dict_filepath"],
            language=data["language"],
            original_form=data["original_form"],
            line_number=data.get("line_number"),
            category=data.get("category", "standard"),
        )


@dataclass
class Word:
    """A normalized word with full metadata."""

    normalized: str                         # Normalized form (lowercase, ASCII)
    length: int                             # Character count
    word_type: str                          # e.g., "5-c"
    sources: list[WordSource] = field(default_factory=list)
    categories: set[str] = field(default_factory=set)
    languages: set[str] = field(default_factory=set)
    tags: set[str] = field(default_factory=set)
    ipa: Optional[str] = None               # IPA transcription (if computed)
    synthesis_groups: set[str] = field(default_factory=set)

    def __post_init__(self):
        """Derive fields from sources."""
        for source in self.sources:
            self.categories.add(source.category)
            self.languages.add(source.language)

    def add_source(self, source: WordSource) -> None:
        """Add a source and update derived fields."""
        self.sources.append(source)
        self.categories.add(source.category)
        self.languages.add(source.language)

    def get_source_dicts(self) -> list[str]:
        """Get list of source dictionary names."""
        return [s.dict_name for s in self.sources]

    def matches_filter(
        self,
        include_categories: Optional[set[str]] = None,
        exclude_categories: Optional[set[str]] = None,
        include_languages: Optional[set[str]] = None,
        min_length: Optional[int] = None,
        max_length: Optional[int] = None,
    ) -> bool:
        """Check if word matches filter criteria."""
        if min_length and self.length < min_length:
            return False
        if max_length and self.length > max_length:
            return False
        if include_languages and not self.languages & include_languages:
            return False
        if include_categories and not self.categories & include_categories:
            return False
        if exclude_categories and self.categories & exclude_categories:
            return False
        return True

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "normalized": self.normalized,
            "length": self.length,
            "type": self.word_type,
            "sources": [s.to_dict() for s in self.sources],
            "categories": sorted(self.categories),
            "languages": sorted(self.languages),
            "tags": sorted(self.tags),
            "ipa": self.ipa,
            "synthesis_groups": sorted(self.synthesis_groups),
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "Word":
        """Create from dictionary."""
        word = cls(
            normalized=data["normalized"],
            length=data["length"],
            word_type=data["type"],
            sources=[WordSource.from_dict(s) for s in data.get("sources", [])],
            ipa=data.get("ipa"),
        )
        word.categories = set(data.get("categories", []))
        word.languages = set(data.get("languages", []))
        word.tags = set(data.get("tags", []))
        word.synthesis_groups = set(data.get("synthesis_groups", []))
        return word


@dataclass
class Dictionary:
    """A collection of words with metadata."""

    name: str                               # e.g., "en_5-c", "en_tr_synthesis"
    language: Optional[str] = None          # Single language or None for synthesis
    languages: set[str] = field(default_factory=set)
    words: dict[str, Word] = field(default_factory=dict)  # normalized -> Word
    generated_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )
    source_dicts: set[str] = field(default_factory=set)
    word_type: Optional[str] = None         # e.g., "5-c" for length-specific dicts

    def add_word(self, word: Word) -> None:
        """Add or merge a word."""
        if word.normalized in self.words:
            existing = self.words[word.normalized]
            for source in word.sources:
                existing.add_source(source)
            existing.tags.update(word.tags)
        else:
            self.words[word.normalized] = word

        self.languages.update(word.languages)
        self.source_dicts.update(word.get_source_dicts())

    def get_words_list(self) -> list[Word]:
        """Get sorted list of words."""
        return sorted(self.words.values(), key=lambda w: w.normalized)

    def count(self) -> int:
        """Get word count."""
        return len(self.words)

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "name": self.name,
            "language": self.language,
            "languages": sorted(self.languages),
            "word_type": self.word_type,
            "generated_at": self.generated_at,
            "source_dicts": sorted(self.source_dicts),
            "word_count": self.count(),
            "words": {
                w.normalized: w.to_dict() for w in self.get_words_list()
            },
        }

    def save(self, filepath: Path) -> None:
        """Save dictionary to JSON file."""
        filepath.parent.mkdir(parents=True, exist_ok=True)
        with open(filepath, "w", encoding="utf-8") as f:
            json.dump(self.to_dict(), f, indent=2, ensure_ascii=False)

    @classmethod
    def load(cls, filepath: Path) -> "Dictionary":
        """Load dictionary from JSON file."""
        with open(filepath, "r", encoding="utf-8") as f:
            data = json.load(f)

        d = cls(
            name=data["name"],
            language=data.get("language"),
            word_type=data.get("word_type"),
            generated_at=data.get("generated_at", ""),
        )
        d.languages = set(data.get("languages", []))
        d.source_dicts = set(data.get("source_dicts", []))

        for word_data in data.get("words", {}).values():
            d.add_word(Word.from_dict(word_data))

        return d
