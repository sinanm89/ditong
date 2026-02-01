"""Base ingestor interface for dictionary sources.

All ingestors inherit from Ingestor and implement the ingest() method.
This provides a consistent API for loading words from any source format.
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional, Iterator
import urllib.request
import os

from ..schema import Word, WordSource, WordType
from ..normalizer import normalize_and_validate


@dataclass
class IngestResult:
    """Result of ingesting a dictionary source."""

    words: list[Word]
    source_path: str
    dict_name: str
    language: str
    category: str
    total_raw: int = 0          # Total lines/entries in source
    total_valid: int = 0        # Valid words after normalization
    total_duplicates: int = 0   # Duplicates within this source
    errors: list[str] = field(default_factory=list)

    def __repr__(self) -> str:
        return (
            f"IngestResult({self.dict_name}: "
            f"{self.total_valid}/{self.total_raw} valid, "
            f"{self.total_duplicates} dupes)"
        )


class Ingestor(ABC):
    """Base class for dictionary ingestors.

    Subclasses must implement:
        - parse(filepath) -> Iterator of (original_word, line_number) tuples
        - file_extensions: list of supported extensions

    The ingest() method handles normalization and Word creation.
    """

    file_extensions: list[str] = []

    def __init__(
        self,
        language: str,
        category: str = "standard",
        min_length: int = 3,
        max_length: int = 10,
    ):
        """Initialize ingestor.

        Args:
            language: Language code (e.g., "en", "tr").
            category: Category tag (e.g., "standard", "urban", "curseword").
            min_length: Minimum word length to include.
            max_length: Maximum word length to include.
        """
        self.language = language
        self.category = category
        self.min_length = min_length
        self.max_length = max_length

    @abstractmethod
    def parse(self, filepath: Path) -> Iterator[tuple[str, Optional[int]]]:
        """Parse source file and yield (word, line_number) tuples.

        Args:
            filepath: Path to source file.

        Yields:
            Tuples of (original_word, line_number).
        """
        pass

    def get_dict_name(self, filepath: Path) -> str:
        """Generate dictionary name from filepath."""
        return filepath.stem

    def ingest(self, filepath: Path | str) -> IngestResult:
        """Ingest dictionary from file.

        Args:
            filepath: Path to source file.

        Returns:
            IngestResult with words and statistics.
        """
        filepath = Path(filepath)
        dict_name = self.get_dict_name(filepath)
        filepath_str = str(filepath.resolve())

        words: dict[str, Word] = {}
        total_raw = 0
        duplicates = 0
        errors: list[str] = []

        for original_word, line_num in self.parse(filepath):
            total_raw += 1

            normalized = normalize_and_validate(original_word)
            if normalized is None:
                continue

            length = len(normalized)
            if length < self.min_length or length > self.max_length:
                continue

            word_type = WordType.from_length(length)
            if word_type is None:
                continue

            source = WordSource(
                dict_name=dict_name,
                dict_filepath=filepath_str,
                language=self.language,
                original_form=original_word,
                line_number=line_num,
                category=self.category,
            )

            if normalized in words:
                words[normalized].add_source(source)
                duplicates += 1
            else:
                words[normalized] = Word(
                    normalized=normalized,
                    length=length,
                    word_type=word_type.value,
                    sources=[source],
                )

        return IngestResult(
            words=list(words.values()),
            source_path=filepath_str,
            dict_name=dict_name,
            language=self.language,
            category=self.category,
            total_raw=total_raw,
            total_valid=len(words),
            total_duplicates=duplicates,
            errors=errors,
        )


class DownloadableIngestor(Ingestor):
    """Ingestor that can download source files."""

    download_urls: dict[str, str] = {}  # language -> URL

    def __init__(
        self,
        language: str,
        cache_dir: Path | str,
        category: str = "standard",
        min_length: int = 3,
        max_length: int = 10,
    ):
        super().__init__(language, category, min_length, max_length)
        self.cache_dir = Path(cache_dir)

    def get_cached_path(self) -> Path:
        """Get path where downloaded file should be cached."""
        if self.language not in self.download_urls:
            raise ValueError(f"No download URL for language: {self.language}")
        url = self.download_urls[self.language]
        filename = url.split("/")[-1]
        return self.cache_dir / filename

    def download(self, force: bool = False) -> Path:
        """Download source file if not cached.

        Args:
            force: Force re-download even if cached.

        Returns:
            Path to cached file.
        """
        if self.language not in self.download_urls:
            raise ValueError(f"No download URL for language: {self.language}")

        url = self.download_urls[self.language]
        cached_path = self.get_cached_path()

        if cached_path.exists() and not force:
            print(f"[{self.language}] Using cached: {cached_path}")
            return cached_path

        self.cache_dir.mkdir(parents=True, exist_ok=True)

        print(f"[{self.language}] Downloading from: {url}")
        urllib.request.urlretrieve(url, cached_path)
        print(f"[{self.language}] Saved to: {cached_path}")

        return cached_path

    def download_and_ingest(self, force: bool = False) -> IngestResult:
        """Download and ingest in one step."""
        filepath = self.download(force=force)
        return self.ingest(filepath)
