"""Hunspell dictionary ingestor.

Parses Hunspell .dic files used by LibreOffice, Firefox, etc.

Format:
    12345           # Optional word count (first line)
    word/FLAGS      # Word with optional affix flags
    another         # Word without flags

Downloads from wooorm/dictionaries (MIT licensed, regularly updated).
"""

from pathlib import Path
from typing import Iterator, Optional

from .base import DownloadableIngestor

# wooorm/dictionaries - MIT licensed, comprehensive
HUNSPELL_URLS = {
    "en": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/en/index.dic",
    "tr": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/tr/index.dic",
    "de": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/de/index.dic",
    "fr": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/fr/index.dic",
    "es": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/es/index.dic",
    "it": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/it/index.dic",
    "pt": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/pt/index.dic",
    "nl": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/nl/index.dic",
    "pl": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/pl/index.dic",
    "ru": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/ru/index.dic",
}


class HunspellIngestor(DownloadableIngestor):
    """Ingestor for Hunspell .dic files."""

    file_extensions = [".dic"]
    download_urls = HUNSPELL_URLS

    def get_dict_name(self, filepath: Path) -> str:
        """Generate dictionary name."""
        return f"hunspell_{self.language}"

    def parse(self, filepath: Path) -> Iterator[tuple[str, Optional[int]]]:
        """Parse Hunspell .dic file.

        Args:
            filepath: Path to .dic file.

        Yields:
            Tuples of (word, line_number).
        """
        with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
            for line_num, line in enumerate(f, start=1):
                line = line.strip()
                if not line:
                    continue

                # Skip first line if it's just a number (word count)
                if line_num == 1 and line.isdigit():
                    continue

                # Strip affix flags: "word/ABC" -> "word"
                if "/" in line:
                    word = line.split("/")[0]
                else:
                    word = line

                if word:
                    yield word, line_num


def ingest(
    filepath: Path | str,
    language: str,
    category: str = "standard",
    min_length: int = 3,
    max_length: int = 10,
):
    """Convenience function to ingest a Hunspell dictionary.

    Args:
        filepath: Path to .dic file.
        language: Language code.
        category: Category tag.
        min_length: Minimum word length.
        max_length: Maximum word length.

    Returns:
        IngestResult with words.
    """
    ingestor = HunspellIngestor(
        language=language,
        cache_dir=Path(filepath).parent,
        category=category,
        min_length=min_length,
        max_length=max_length,
    )
    return ingestor.ingest(filepath)


def download_and_ingest(
    language: str,
    cache_dir: Path | str,
    category: str = "standard",
    min_length: int = 3,
    max_length: int = 10,
    force: bool = False,
):
    """Download and ingest a Hunspell dictionary.

    Args:
        language: Language code (en, tr, de, etc.).
        cache_dir: Directory to cache downloaded files.
        category: Category tag.
        min_length: Minimum word length.
        max_length: Maximum word length.
        force: Force re-download.

    Returns:
        IngestResult with words.
    """
    ingestor = HunspellIngestor(
        language=language,
        cache_dir=cache_dir,
        category=category,
        min_length=min_length,
        max_length=max_length,
    )
    return ingestor.download_and_ingest(force=force)


def get_supported_languages() -> list[str]:
    """Return list of languages with available Hunspell downloads."""
    return list(HUNSPELL_URLS.keys())
